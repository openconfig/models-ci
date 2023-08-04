package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/openconfig/models-ci/commonci"
)

type versionRecord struct {
	File            string
	OldMajorVersion uint64
	NewMajorVersion uint64
	OldVersion      string
	NewVersion      string
}

type versionRecordSlice []versionRecord

func (s versionRecordSlice) MajorVersionChanges() string {
	if len(s) == 0 {
		return fmt.Sprintf("No major YANG version changes in commit %s", commitSHA)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Major YANG version changes in commit %s:\n", commitSHA))
	for _, change := range s {
		if change.OldMajorVersion != change.NewMajorVersion {
			b.WriteString(fmt.Sprintf("%s: `%s` -> `%s`\n", change.File, change.OldVersion, change.NewVersion))
		}
	}
	return b.String()
}

func (s versionRecordSlice) hasBreaking() bool {
	for _, change := range s {
		if change.OldMajorVersion != 0 && change.OldMajorVersion != change.NewMajorVersion {
			return true
		}
	}
	return false
}

// processMiscChecksOutput takes the raw result output from the misc-checks
// results directory and returns its formatted report and pass/fail status.
//
// It also returns a list of version changes for each file.
func processMiscChecksOutput(resultsDir string) (string, bool, versionRecordSlice, error) {
	fileProperties := map[string]map[string]string{}
	changedFiles, err := readYangFilesList(filepath.Join(resultsDir, "changed-files.txt"))
	if err != nil {
		return "", false, nil, err
	}
	for _, file := range changedFiles {
		if _, ok := fileProperties[file]; !ok {
			fileProperties[file] = map[string]string{}
		}
		fileProperties[file]["changed"] = "true"
	}
	if err := readGoyangVersionsLog(filepath.Join(resultsDir, "pr-file-parse-log"), false, fileProperties); err != nil {
		return "", false, nil, err
	}
	if err := readGoyangVersionsLog(filepath.Join(resultsDir, "master-file-parse-log"), true, fileProperties); err != nil {
		return "", false, nil, err
	}

	var ocVersionViolations []string
	ocVersionChangedCount := 0
	var reachabilityViolations []string
	filesReachedCount := 0
	// Only look at the PR's files as they might be different from the master's files.
	allNonEmptyPRFiles, err := readYangFilesList(filepath.Join(resultsDir, "all-non-empty-files.txt"))
	if err != nil {
		return "", false, nil, err
	}
	moduleFileGroups := map[string][]fileAndVersion{}
	var versionRecords versionRecordSlice
	for _, file := range allNonEmptyPRFiles {
		properties, ok := fileProperties[file]

		// Reachability check
		if !ok || properties["reachable"] != "true" {
			reachabilityViolations = append(reachabilityViolations, sprintLineHTML("%s: file not used by any .spec.yml build.", file))
			// If the file was not reached, then its other
			// parameters would not have been parsed by goyang, so
			// simply skip the rest of the checks.
			continue
		}
		filesReachedCount += 1

		// openconfig-version update check
		ocVersion, hasVersion := properties["openconfig-version"]
		masterOcVersion, hadVersion := properties["master-openconfig-version"]
		switch {
		case properties["changed"] != "true":
			// We assume the versioning is correct without change.
		case hadVersion && hasVersion:
			oldver, newver, err := checkSemverIncrease(masterOcVersion, ocVersion, "openconfig-version")
			if err != nil {
				ocVersionViolations = append(ocVersionViolations, sprintLineHTML(file+": "+err.Error()))
				break
			}
			ocVersionChangedCount += 1
			versionRecords = append(versionRecords, versionRecord{
				File:            file,
				OldMajorVersion: oldver.Major(),
				NewMajorVersion: newver.Major(),
				OldVersion:      masterOcVersion,
				NewVersion:      ocVersion,
			})
		case hadVersion && !hasVersion:
			ocVersionViolations = append(ocVersionViolations, sprintLineHTML("%s: openconfig-version was removed", file))
		default: // If didn't have version before, any new version is accepted.
			ocVersionChangedCount += 1
		}

		if mod, ok := properties["belonging-module"]; hasVersion && ok {
			// Error checking is already done by the version update check.
			if v, err := semver.StrictNewVersion(ocVersion); err == nil {
				moduleFileGroups[mod] = append(moduleFileGroups[mod], fileAndVersion{name: file, version: v})
			}
		}
	}

	// Compute HTML string and pass/fail status.
	var out strings.Builder
	var pass = true
	appendViolationOut := func(desc string, violations []string, passString string) {
		if len(violations) == 0 {
			out.WriteString(sprintSummaryHTML(commonci.BoolStatusToString(true), desc, passString))
		} else {
			out.WriteString(sprintSummaryHTML(commonci.BoolStatusToString(false), desc, strings.Join(violations, "")))
			pass = false
		}
	}
	appendViolationOut("openconfig-version update check", ocVersionViolations, fmt.Sprintf("%d file(s) correctly updated.\n", ocVersionChangedCount))
	appendViolationOut(".spec.yml build reachability check", reachabilityViolations, fmt.Sprintf("%d files reached by build rules.\n", filesReachedCount))
	appendViolationOut("submodule versions must match the belonging module's version", versionGroupViolationsHTML(moduleFileGroups), fmt.Sprintf("%d module/submodule file groups have matching versions", len(moduleFileGroups)))

	return out.String(), pass, versionRecords, nil
}
