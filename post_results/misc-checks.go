// Copyright 2023 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
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

// readYangFilesList reads a file containing a list of YANG files, and returns
// a slice of these files. An unrecognized line causes an error to be returned.
// The error checking is not robust, but should be sufficient for our limited use.
func readYangFilesList(path string) ([]string, error) {
	filesStr, err := readFile(path)
	if err != nil {
		return nil, err
	}

	fileMap := map[string]bool{}
	for _, line := range strings.Split(filesStr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fileSegments := strings.Split(line, "/")
		yangFileName := strings.TrimSpace(fileSegments[len(fileSegments)-1])
		if !strings.HasSuffix(yangFileName, ".yang") {
			return nil, fmt.Errorf("while parsing %s: unrecognized line, expected a path ending in a YANG file: %s", path, line)
		}
		fileMap[yangFileName] = true
	}

	var files []string
	for f := range fileMap {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// readGoyangVersionsLog returns a map of YANG files to file attributes as parsed from the log.
// The file should be a list of YANG file to space-separated attributes.
// e.g.
// foo.yang: openconfig-version:"1.2.3" revision-version:"2.3.4"
func readGoyangVersionsLog(logPath string, masterBranch bool, fileProperties map[string]map[string]string) error {
	fileLog, err := readFile(logPath)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(fileLog, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fileSegments := strings.SplitN(line, ":", 2)
		yangFileName := strings.TrimSpace(fileSegments[0])
		if !strings.HasSuffix(yangFileName, ".yang") {
			return fmt.Errorf("while parsing %s: unrecognized line heading %q, expected a \"<name>.yang:\" start to the line: %q", logPath, yangFileName, line)
		}
		propertyMap, ok := fileProperties[yangFileName]
		if !ok {
			propertyMap = map[string]string{}
			fileProperties[yangFileName] = propertyMap
		}

		if !masterBranch {
			propertyMap["reachable"] = "true"
		}

		for _, property := range strings.Fields(strings.TrimSpace(fileSegments[1])) {
			segments := strings.SplitN(property, ":", 2)
			if len(segments) != 2 {
				return fmt.Errorf("while parsing %s: unrecognized property substring, expected \"<property name>:\"<property>\"\" separated by spaces: %q", logPath, property)
			}
			name, value := segments[0], segments[1]
			if value[0] == '"' {
				if len(value) == 1 || value[len(value)-1] != '"' {
					return fmt.Errorf("while parsing %s: Got invalid property value format: %s -- if the property value starts with a quote, it is assumed to be an enclosing quote", logPath, property)
				}
				value = value[1 : len(value)-1] // Remove enclosing quotes.
			}
			switch name {
			case "openconfig-version", "belonging-module", "latest-revision-version":
				if masterBranch {
					name = "master-" + name
				}
				propertyMap[name] = value
			default:
				log.Printf("skipped unrecognized YANG file property: %s", property)
			}
		}
	}
	return nil
}

// checkSemverIncrease checks that newVersion is greater than the oldVersion
// according to semantic versioning rules.
// Note that any increase is fine, including jumps, e.g. 1.0.0 -> 1.0.2.
// If there isn't an increase, a descriptive error message is returned.
func checkSemverIncrease(oldVersion, newVersion, versionStringName string) (*semver.Version, *semver.Version, error) {
	newV, err := semver.StrictNewVersion(newVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid version string: %q", newVersion)
	}
	oldV, err := semver.StrictNewVersion(oldVersion)
	switch {
	case err != nil:
		return nil, nil, fmt.Errorf("unexpected error, base branch version string unparseable: %q", oldVersion)
	case newV.Equal(oldV):
		return nil, nil, fmt.Errorf("file updated but %s string not updated: %q", versionStringName, oldVersion)
	case !newV.GreaterThan(oldV):
		return nil, nil, fmt.Errorf("new semantic version not valid, old version: %q, new version: %q", oldVersion, newVersion)
	default:
		return oldV, newV, nil
	}
}

type fileAndVersion struct {
	name    string
	version *semver.Version
}

// versionGroupViolationsHTML returns the version violations where a group of
// module/submodule files don't have matching versions.
func versionGroupViolationsHTML(moduleFileGroups map[string][]fileAndVersion) []string {
	var violations []string

	var modules []string
	for m := range moduleFileGroups {
		modules = append(modules, m)
	}
	sort.Strings(modules)
	for _, moduleName := range modules {
		latestVersion := semver.MustParse("0.0.0")
		latestVersionModule := ""
		for _, nameAndVersion := range moduleFileGroups[moduleName] {
			if nameAndVersion.version.GreaterThan(latestVersion) {
				latestVersion = nameAndVersion.version
				latestVersionModule = nameAndVersion.name
			}
		}
		latestVersionString := latestVersion.Original()

		var violation strings.Builder
		for _, nameAndVersion := range moduleFileGroups[moduleName] {
			if version := nameAndVersion.version.Original(); version != latestVersionString {
				if violation.Len() != 0 {
					violation.WriteString(",")
				}
				violation.WriteString(fmt.Sprintf(" <b>%s</b> (%s)", nameAndVersion.name, version))
			}
		}
		if violation.Len() != 0 {
			violations = append(violations, sprintLineHTML("module set %s is at <b>%s</b> (%s), non-matching files:%s", moduleName, latestVersionString, latestVersionModule, violation.String()))
		}
	}
	return violations
}
