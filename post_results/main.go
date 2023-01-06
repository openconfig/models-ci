// Copyright 2020 Google Inc.
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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"log"

	"github.com/Masterminds/semver/v3"
	"github.com/openconfig/models-ci/commonci"
	"github.com/openconfig/models-ci/util"
)

// post_results posts the CI results for a given tool using the output from
// running the tool's script generated by cmd_gen. The location of the results
// is determined by common_ci.

const (
	// IgnorePyangWarnings ignores all warnings from pyang or pyang-based tools.
	IgnorePyangWarnings = true
	// IgnoreConfdWarnings ignores all warnings from ConfD.
	IgnoreConfdWarnings = false
	// bucketName is the Google storage bucket name.
	bucketName = "openconfig"
)

var (
	// flags: should be string if it may not exist.
	validatorId string // validatorId is the unique name identifying the validator (see commonci for all of them)
	modelRoot   string // modelRoot is the root directory of the models.
	repoSlug    string // repoSlug is the "owner/repo" name of the models repo (e.g. openconfig/public).
	prNumberStr string // prNumberStr is the PR number.
	branchName  string // branchName is the name of the branch where the commit occurred.
	commitSHA   string
	version     string // version is a specific version of the validator that's being run (empty means latest).

	// derived flags
	owner    string
	repo     string
	prNumber int

	// badgeCmdTemplate is the badge creation and upload command generated for pushes to the master branch.
	badgeCmdTemplate = mustTemplate("badgeCmd", fmt.Sprintf(`REMOTE_PATH_PFX=gs://%s/compatibility-badges/{{ .RepoPrefix }}:
RESULTSDIR={{ .ResultsDir }}
upload-public-file() {
	gsutil cp $RESULTSDIR/$1 "$REMOTE_PATH_PFX"$1
	gsutil acl ch -u AllUsers:R "$REMOTE_PATH_PFX"$1
	gsutil setmeta -h "Cache-Control:no-cache" "$REMOTE_PATH_PFX"$1
}
badge "{{ .Status }}" "{{ .ValidatorDesc }}" :{{ .Colour }} > $RESULTSDIR/{{ .ValidatorAndVersion }}.svg
upload-public-file {{ .ValidatorAndVersion }}.svg
upload-public-file {{ .ValidatorAndVersion }}.html
`, bucketName))
)

// mustTemplate generates a template.Template for a particular named source template
func mustTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Parse(src))
}

// badgeCmdParams is the input to the badge template.
type badgeCmdParams struct {
	RepoPrefix          string
	Status              string
	ValidatorAndVersion string
	ValidatorDesc       string
	Colour              string
	ResultsDir          string
}

func init() {
	flag.StringVar(&validatorId, "validator", "", "unique name of the validator")
	flag.StringVar(&modelRoot, "modelRoot", "", "root directory to OpenConfig models")
	flag.StringVar(&repoSlug, "repo-slug", "", "repo where CI is run")
	flag.StringVar(&prNumberStr, "pr-number", "", "PR number")
	flag.StringVar(&branchName, "branch", "", "branch name of commit")
	flag.StringVar(&commitSHA, "commit-sha", "", "commit SHA of the PR")
	flag.StringVar(&version, "version", "", "(optional) specific version of the validator tool.")
}

// sprintLineHTML prints a single list item to be put under a top-level summary item.
func sprintLineHTML(format string, a ...interface{}) string {
	return fmt.Sprintf("  <li>"+format+"</li>\n", a...)
}

// sprintSummaryHTML prints a top-level summary item containing free-form or list items.
func sprintSummaryHTML(status, title, format string, a ...interface{}) string {
	return fmt.Sprintf("<details>\n  <summary>%s&nbsp; %s</summary>\n"+format+"</details>\n", append([]interface{}{commonci.Emoji(status), title}, a...)...)
}

// readFile reads the entire file into a string and returns it along with an error if any.
func readFile(path string) (string, error) {
	outBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file at path %q: %v\n", path, err)
	}
	return string(outBytes), nil
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

// processMiscChecksOutput takes the raw result output from the misc-checks
// results directory and returns its formatted report and pass/fail status.
func processMiscChecksOutput(resultsDir string) (string, bool, string, error) {
	fileProperties := map[string]map[string]string{}
	changedFiles, err := readYangFilesList(filepath.Join(resultsDir, "changed-files.txt"))
	if err != nil {
		return "", false, "", err
	}
	for _, file := range changedFiles {
		if _, ok := fileProperties[file]; !ok {
			fileProperties[file] = map[string]string{}
		}
		fileProperties[file]["changed"] = "true"
	}
	if err := readGoyangVersionsLog(filepath.Join(resultsDir, "pr-file-parse-log"), false, fileProperties); err != nil {
		return "", false, "", err
	}
	if err := readGoyangVersionsLog(filepath.Join(resultsDir, "master-file-parse-log"), true, fileProperties); err != nil {
		return "", false, "", err
	}

	var ocVersionViolations []string
	ocVersionChangedCount := 0
	var reachabilityViolations []string
	filesReachedCount := 0
	// Only look at the PR's files as they might be different from the master's files.
	allNonEmptyPRFiles, err := readYangFilesList(filepath.Join(resultsDir, "all-non-empty-files.txt"))
	if err != nil {
		return "", false, "", err
	}
	moduleFileGroups := map[string][]fileAndVersion{}
	var majorVersionChanges strings.Builder
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
			} else {
				ocVersionChangedCount += 1
			}
			if oldver != nil && newver != nil {
				if oldver.Major() != newver.Major() {
					majorVersionChanges.WriteString(fmt.Sprintf("%s: `%s` -> `%s`\n", file, masterOcVersion, ocVersion))
				}
			}
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

	return out.String(), pass, majorVersionChanges.String(), nil
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

// processStandardOutput takes raw pyang/confd output and transforms it to an
// HTML format for display on a GitHub gist comment.
// Errors are displayed in front of warnings.
func processStandardOutput(rawOut string, pass, noWarnings bool) (string, error) {
	standardOutput := util.ParseStandardOutput(rawOut)

	var errorLines, nonErrorLines strings.Builder
	for _, errLine := range append(standardOutput.ErrorLines, standardOutput.WarningLines...) {
		// Convert file path to relative path.
		var err error
		if errLine.Path, err = filepath.Rel(modelRoot, errLine.Path); err != nil {
			return "", fmt.Errorf("failed to calculate relpath at path %q (modelRoot %q) parsed from error message: %v\n", errLine.Path, modelRoot, err)
		}

		processedLine := fmt.Sprintf("%s (%d): %s: <pre>%s</pre>", errLine.Path, errLine.LineNo, errLine.Status, errLine.Message)
		switch {
		case strings.Contains(errLine.Status, "error"):
			errorLines.WriteString(sprintLineHTML(processedLine))
		case strings.Contains(errLine.Status, "warning"):
			if !noWarnings {
				nonErrorLines.WriteString(sprintLineHTML(processedLine))
			}
		}
	}
	for _, line := range standardOutput.OtherLines {
		nonErrorLines.WriteString(sprintLineHTML(line))
	}

	var out strings.Builder
	if pass {
		out.WriteString("Passed.\n")
	}
	if errorLines.Len() > 0 || nonErrorLines.Len() > 0 {
		out.WriteString("<ul>\n")
		out.WriteString(errorLines.String())
		out.WriteString(nonErrorLines.String())
		out.WriteString("</ul>\n")
	}
	return out.String(), nil
}

// processPyangOutput takes raw pyang/confd output and transforms it to an
// HTML format for display on a GitHub gist comment.
// Errors are displayed in front of warnings.
func processPyangOutput(rawOut string, pass, noWarnings bool) (string, error) {
	var errorLines, nonErrorLines strings.Builder
	if pyangOutput, err := util.ParsePyangTextprotoOutput(rawOut); err != nil {
		log.Printf("INFO: could not parse pyang output as textproto (raw output below): %v\n%s", err, rawOut)
		nonErrorLines.WriteString(fmt.Sprintf("  <pre>%s</pre>\n", strings.TrimSpace(rawOut)))
	} else {
		for _, msgLine := range pyangOutput.Messages {
			// Convert file path to relative path.
			var err error
			if msgLine.Path, err = filepath.Rel(modelRoot, msgLine.Path); err != nil {
				return "", fmt.Errorf("failed to calculate relpath at path %q (modelRoot %q) parsed from error message: %v\n", msgLine.Path, modelRoot, err)
			}

			processedLine := fmt.Sprintf("%s (%d): %s: <pre>%s</pre>", msgLine.Path, msgLine.Line, msgLine.Type, msgLine.Message)
			switch {
			case strings.Contains(msgLine.Type, "error"):
				errorLines.WriteString(sprintLineHTML(processedLine))
			case strings.Contains(msgLine.Type, "warning"):
				if !noWarnings {
					nonErrorLines.WriteString(sprintLineHTML(processedLine))
				}
			}
		}
	}

	var out strings.Builder
	if pass {
		out.WriteString("Passed.\n")
	}
	if errorLines.Len() > 0 || nonErrorLines.Len() > 0 {
		out.WriteString("<ul>\n")
		out.WriteString(errorLines.String())
		out.WriteString(nonErrorLines.String())
		out.WriteString("</ul>\n")
	}
	return out.String(), nil
}

// userfyBashCommand changes the bash command displayed to the user to be
// something that's easier to use.
func userfyBashCommand(cmd string) string {
	return strings.NewReplacer("/workspace/", "$OC_WORKSPACE/", "$OCPYANG_PLUGIN_DIR", "$GOPATH/src/github.com/openconfig/oc-pyang/openconfig_pyang/plugins", "$PYANGBIND_PLUGIN_DIR", "$GOPATH/src/github.com/robshakir/pyangbind/pyangbind/plugin").Replace(cmd)
}

// parseModelResultsHTML transforms the output files of the validator script into HTML
// to be displayed on GitHub.
// If condensed=true, then only errors are provided.
func parseModelResultsHTML(validatorId, validatorResultDir string, condensed bool) (string, bool, error) {
	var htmlOut, modelHTML strings.Builder
	var prevModelDirName string

	// Used to cache bash command for output.
	var bashCommand string
	var bashCommandModelDirName string
	var bashCommandModelName string

	allPass := true
	modelDirPass := true
	// Process each result file in lexical order.
	// Since result files are in "modelDir==model==status" format, this ensures we're processing by directory.
	// (Note that each modelDir has multiple models. Each model corresponds to a result file).
	if err := filepath.Walk(validatorResultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("handle failure accessing a path %q: %v\n", path, err)
		}

		components := strings.Split(info.Name(), "==")
		// Handle per-model output. Files should be in "modelDir==model==status" format; otherwise they're ignored.
		if !info.IsDir() && len(components) == 3 {
			modelDirName, modelName, status := components[0], components[1], components[2]

			// Write results one modelDir at a time in order to report overall modelDir status.
			if prevModelDirName != "" && modelDirName != prevModelDirName {
				if !condensed || !modelDirPass {
					htmlOut.WriteString(sprintSummaryHTML(commonci.BoolStatusToString(modelDirPass), prevModelDirName, modelHTML.String()))
				}
				modelHTML.Reset()
				modelDirPass = true
			}
			prevModelDirName = modelDirName

			// Get output string.
			outString, err := readFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file at path %q: %v\n", path, err)
			}

			modelPass := true
			switch status {
			case "cmd":
				// Don't do anything, store the command for later output.
				// Since filepath.Walk walks files in lexical
				// order, ${prefix}cmd should be walked first,
				// such that ${prefix}pass or ${prefix}fail
				// will have it ready to display to the user.
				bashCommand = userfyBashCommand(outString)
				bashCommandModelDirName = modelDirName
				bashCommandModelName = modelName
				return nil
			case "pass":
			case "fail":
				allPass = false
				modelDirPass = false
				modelPass = false
			default:
				return fmt.Errorf("expect status at path %q to be true or false, got %v", path, status)
			}

			// Transform output string into HTML.
			switch {
			case strings.Contains(validatorId, "pyang"):
				outString, err = processPyangOutput(outString, modelPass, IgnorePyangWarnings)
			case validatorId == "confd":
				outString, err = processStandardOutput(outString, modelPass, IgnoreConfdWarnings)
			default:
				outString = strings.Join(strings.Split(outString, "\n"), "<br>\n")
				if modelPass {
					outString = "Passed.\n" + outString
				}
			}
			if !modelPass && outString == "" {
				outString = "Failed.\n"
			}
			if err != nil {
				return fmt.Errorf("error encountered while processing output for validator %q: %v", validatorId, err)
			}

			if !condensed || !modelPass {
				// Display bash command that produced the validator result.
				var bashCommandSummary string
				if bashCommand != "" && bashCommandModelDirName == modelDirName && bashCommandModelName == modelName {
					bashCommandSummary = fmt.Sprintf("%s&nbsp; %s\n<pre>%s</pre>\n", commonci.Emoji("cmd"), "bash command", bashCommand)
				}
				modelHTML.WriteString(sprintSummaryHTML(status, modelName, bashCommandSummary+outString))
			}
		}
		return nil
	}); err != nil {
		return "", false, err
	}

	// Edge case: handle last modelDir.
	if !condensed || !modelDirPass {
		htmlOut.WriteString(sprintSummaryHTML(commonci.BoolStatusToString(modelDirPass), prevModelDirName, modelHTML.String()))
	}

	return htmlOut.String(), allPass, nil
}

// getResult parses the results for the given validator and its results
// directory, and returns the string to be put in a GitHub gist comment as well
// as the status (i.e. pass or fail), and whether the changes are
// backward-compatible.
// If condensed=true, then only errors are provided.
func getResult(validatorId, resultsDir string, condensed bool) (string, bool, string, error) {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return "", false, "", fmt.Errorf("validator %q not found!", validatorId)
	}

	// outString is parsed stdout.
	var outString string
	// pass is the overall validation result.
	var pass bool
	// majorVersionChanges is a GitHub-formatted string listing major YANG version changes.
	var majorVersionChanges string

	failFileBytes, err := ioutil.ReadFile(filepath.Join(resultsDir, commonci.FailFileName))
	// existent fail file == failure.
	executionFailed := err == nil
	err = nil

	switch {
	case executionFailed:
		outString = string(failFileBytes)
		if outString == "" {
			outString = "Test failed with no stderr output."
		}
		// For per-model validators, an execution failure suggests a CI infra failure.
		if validator.IsPerModel {
			outString = "Validator script failed -- infra bug?\n" + outString
		}
		pass = false
	case validator.IsPerModel && validatorId == "misc-checks":
		outString, pass, majorVersionChanges, err = processMiscChecksOutput(resultsDir)
	case validator.IsPerModel:
		outString, pass, err = parseModelResultsHTML(validatorId, resultsDir, condensed)
		if pass && condensed {
			outString = "All passed.\n" + outString
		}
	default:
		outString = "Test passed."
		pass = true
	}

	return outString, pass, majorVersionChanges, err
}

// WriteBadgeUploadCmdFile writes a bash script into resultsDir that posts a
// status badge for the given validator and result into cloud storage.
func WriteBadgeUploadCmdFile(validatorDesc, validatorUniqueStr string, pass bool, resultsDir string) (string, error) {
	// Badge creation and upload command.
	var builder strings.Builder
	status := "fail"
	colour := "red"
	if pass {
		status = "pass"
		colour = "brightgreen"
	}
	if err := badgeCmdTemplate.Execute(&builder, &badgeCmdParams{
		RepoPrefix:          strings.ReplaceAll(repoSlug, "/", "-"), // Make repo slug safe for use as file name.
		Status:              status,
		ValidatorAndVersion: validatorUniqueStr,
		ValidatorDesc:       validatorDesc,
		Colour:              colour,
		ResultsDir:          resultsDir,
	}); err != nil {
		return "", err
	}

	return builder.String(), nil
}

// getGistHeading gets the description and content of the result gist for the
// given validator from its script output file. The "description" is the title
// of the gist, and "content" is the script execution output.
// NOTE: The parsed test result output (distinct from the script execution
// output) should be attached as a comment on the same gist.
func getGistHeading(validatorId, version, resultsDir string) (string, string, error) {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return "", "", fmt.Errorf("getGistHeading: validator %q not found!", validatorId)
	}

	validatorDesc := validator.StatusName(version)
	// If version is latest, then get the concrete version output by the tool if it exists.
	if version == "" {
		if outBytes, err := ioutil.ReadFile(filepath.Join(resultsDir, commonci.LatestVersionFileName)); err != nil {
			log.Printf("INFO: did not read latest version for %s: %v", validatorId, err)
		} else {
			// Get the first line of the version output as the tool's display title.
			nameAndVersionParts := strings.Fields(strings.TrimSpace(strings.SplitN(string(outBytes), "\n", 2)[0]))
			// Format it a little.
			validatorDesc = nameAndVersionParts[0]
			if len(nameAndVersionParts) > 1 {
				validatorDesc = commonci.AppendVersionToName(validatorDesc, strings.Join(nameAndVersionParts[1:], " "))
			}
		}
	}

	outBytes, err := ioutil.ReadFile(filepath.Join(resultsDir, commonci.OutFileName))
	if err != nil {
		return "", "", err
	}
	content := string(outBytes)
	if content == "" {
		content = "No output"
	}

	return validatorDesc, content, nil
}

// postCompatibilityReport posts the results for the validators to be reported
// under a compatibility report.
func postCompatibilityReport(validatorAndVersions []commonci.ValidatorAndVersion) error {
	if len(validatorAndVersions) == 0 {
		log.Printf("Skipping compatibility report -- no validator to report.")
		return nil
	}

	validator, ok := commonci.Validators["compat-report"]
	if !ok {
		return fmt.Errorf("CI infra failure: compatibility report validator not found in commonci.Validators")
	}

	// Get the combined execution output, as well as each validator's header description.
	var executionOutput string
	var validatorDescs []string
	for _, vv := range validatorAndVersions {
		resultsDir := commonci.ValidatorResultsDir(vv.ValidatorId, vv.Version)

		validatorDesc, content, err := getGistHeading(vv.ValidatorId, vv.Version, resultsDir)
		if err != nil {
			return fmt.Errorf("postResult: %v", err)
		}
		executionOutput += validatorDesc + ":\n" + content + "\n"
		validatorDescs = append(validatorDescs, validatorDesc)
	}

	// Post the gist to contain each validator's results.
	var g *commonci.GithubRequestHandler
	var err error
	var gistURL, gistID string
	if err := commonci.Retry(5, "CreateCIOutputGist", func() error {
		g, err = commonci.NewGitHubRequestHandler()
		if err != nil {
			return err
		}
		gistURL, gistID, err = g.CreateCIOutputGist(validator.Name, executionOutput)
		return err
	}); err != nil {
		return fmt.Errorf("postResult: couldn't create gist: %v", err)
	}

	// Post a gist comment for each validator.
	// Also, build a PR comment to be posted on the PR page linking to each gist comment.
	var commentBuilder strings.Builder
	commentBuilder.WriteString(fmt.Sprintf("Compatibility Report for commit %s:\n", commitSHA))
	for i, vv := range validatorAndVersions {
		resultsDir := commonci.ValidatorResultsDir(vv.ValidatorId, vv.Version)

		// Post parsed test results as a gist comment.
		testResultString, pass, _, err := getResult(vv.ValidatorId, resultsDir, false)
		if err != nil {
			return fmt.Errorf("postResult: couldn't parse results for <%s>@<%s> in resultsDir %q: %v", vv.ValidatorId, vv.Version, resultsDir, err)
		}

		gistTitle := fmt.Sprintf("%s %s", commonci.Emoji(commonci.BoolStatusToString(pass)), validatorDescs[i])
		id, err := g.AddGistComment(gistID, gistTitle, testResultString)
		if err != nil {
			return fmt.Errorf("postResult: could not add gist comment: %v", err)
		}

		commentBuilder.WriteString(fmt.Sprintf("%s [%s](%s#gistcomment-%d)\n", commonci.Emoji(commonci.BoolStatusToString(pass)), validatorDescs[i], gistURL, id))
	}
	comment := commentBuilder.String()
	if err := g.AddEditOrDeletePRComment("Compatibility Report for commit", &comment, owner, repo, prNumber); err != nil {
		return fmt.Errorf("postCompatibilityReport: couldn't post comment: %v", err)
	}
	return nil
}

// postResult retrieves the test output for the given validator and version
// from its results folder and posts a gist and PR status linking to the gist.
func postResult(validatorId, version string) error {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return fmt.Errorf("postResult: validator %q not found!", validatorId)
	}
	resultsDir := commonci.ValidatorResultsDir(validatorId, version)

	pushToMaster := false
	// If it's a push on master, just upload badge for normal validators as the only action.
	if prNumber == 0 {
		if branchName != "master" {
			return fmt.Errorf("postResult: There is no action to take for a non-master branch push, please re-examine your push triggers")
		}
		pushToMaster = true
	}

	compatReportsStr, err := readFile(commonci.CompatReportValidatorsFile)
	if err != nil {
		return fmt.Errorf("postResult: %v", err)
	}
	compatValidators, compatValidatorsMap := commonci.GetValidatorAndVersionsFromString(compatReportsStr)

	if !pushToMaster {
		if validatorId == "compat-report" {
			log.Printf("Processing compatibility report for %s", compatReportsStr)
			return postCompatibilityReport(compatValidators)
		}

		// Skip PR status reporting if validator is part of compatibility report.
		if compatValidatorsMap[validatorId][version] {
			log.Printf("Validator %s part of compatibility report, skipping reporting standalone PR status.", commonci.AppendVersionToName(validatorId, version))
			return nil
		}
	}

	// Get information needed for posting badge or GitHub gist.
	validatorDesc, runOutput, err := getGistHeading(validatorId, version, resultsDir)
	if err != nil {
		return fmt.Errorf("postResult: %v", err)
	}
	testResultString, pass, majorVersionChanges, err := getResult(validatorId, resultsDir, false)
	if err != nil {
		return fmt.Errorf("postResult: couldn't parse results: %v", err)
	}

	if pushToMaster {
		if validator.ReportOnly {
			// Only upload results for running validators.
			return nil
		}
		// Output badge creation & upload commands into a file to be executed.
		validatorUniqueStr := commonci.AppendVersionToName(validatorId, version)
		uploadCmdFileContent, err := WriteBadgeUploadCmdFile(validatorDesc, validatorUniqueStr, pass, resultsDir)
		if err != nil {
			return fmt.Errorf("postResult: couldn't upload badge command for <%s>@<%s> in resultsDir %q: %v", validatorId, version, resultsDir, err)
		}
		badgeUploadFile := filepath.Join(resultsDir, commonci.BadgeUploadCmdFile)
		if err := ioutil.WriteFile(badgeUploadFile, []byte(uploadCmdFileContent), 0444); err != nil {
			log.Fatalf("error while writing validator pass file %q: %v", badgeUploadFile, err)
			return err
		}

		// Put output into a file to be uploaded and linked by the badges.
		outputHTML := fmt.Sprintf("<p>%s</p><span style=\"white-space: pre-line\"><p>Execution output:\n%s</p></span>", testResultString, runOutput)
		outputFile := filepath.Join(resultsDir, validatorUniqueStr+".html")
		if err := ioutil.WriteFile(outputFile, []byte(outputHTML), 0666); err != nil {
			log.Fatalf("error while writing output file %q: %v", outputFile, err)
			return err
		}

		// Skip PR status reporting if validator is part of compatibility report.
		if compatValidatorsMap[validatorId][version] {
			log.Printf("Validator %s part of compatibility report, skipping reporting standalone PR status.", commonci.AppendVersionToName(validatorId, version))
			return nil
		}
	}

	var url, gistID string
	var g *commonci.GithubRequestHandler

	// Create gist representing test results. The "validatorDesc" is the
	// title of the gist, and "runOutput" is the script execution output.
	if err := commonci.Retry(5, "CreateCIOutputGist", func() error {
		g, err = commonci.NewGitHubRequestHandler()
		if err != nil {
			return err
		}
		url, gistID, err = g.CreateCIOutputGist(validatorDesc, runOutput)
		return err
	}); err != nil {
		return fmt.Errorf("postResult: couldn't create gist: %v", err)
	}

	if !pushToMaster && validatorId == "misc-checks" {
		var majorVersionChangesComment string
		switch majorVersionChanges {
		case "":
			majorVersionChangesComment = fmt.Sprintf("No major YANG version changes in commit %s", commitSHA)
			if err := g.PostLabel("non-breaking", "00FF00", owner, repo, prNumber); err != nil {
				return fmt.Errorf("couldn't post label: %v", err)
			}
			if err := g.DeleteLabel("breaking", owner, repo, prNumber); err != nil {
				return fmt.Errorf("couldn't delete label: %v", err)
			}
		default:
			majorVersionChangesComment = fmt.Sprintf("Major YANG version changes in commit %s:\n%s", commitSHA, majorVersionChanges)
			if err := g.PostLabel("breaking", "FF0000", owner, repo, prNumber); err != nil {
				return fmt.Errorf("couldn't post label: %v", err)
			}
			if err := g.DeleteLabel("non-breaking", owner, repo, prNumber); err != nil {
				return fmt.Errorf("couldn't delete label: %v", err)
			}
		}
		if err := g.AddEditOrDeletePRComment("Major YANG version changes in commit", &majorVersionChangesComment, owner, repo, prNumber); err != nil {
			return fmt.Errorf("couldn't post Major YANG version changes comment: %v", err)
		}
	}

	// Post parsed test results as a gist comment.
	if _, err := g.AddGistComment(gistID, fmt.Sprintf("%s %s", commonci.Emoji(commonci.BoolStatusToString(pass)), validatorDesc), testResultString); err != nil {
		return fmt.Errorf("postResult: could not add gist comment: %v", err)
	}

	prUpdate := &commonci.GithubPRUpdate{
		Owner:   owner,
		Repo:    repo,
		Ref:     commitSHA,
		URL:     url,
		Context: validator.StatusName(version),
	}
	if pass {
		prUpdate.NewStatus = "success"
		prUpdate.Description = validatorDesc + " Succeeded"
	} else {
		prUpdate.NewStatus = "failure"
		prUpdate.Description = validatorDesc + " Failed"
	}

	if uperr := g.UpdatePRStatus(prUpdate); uperr != nil {
		return fmt.Errorf("postResult: couldn't update PR: %s", uperr)
	}
	return nil
}

func main() {
	flag.Parse()
	if repoSlug == "" {
		log.Fatalf("no repo slug input")
	}
	repoSplit := strings.Split(repoSlug, "/")
	owner = repoSplit[0]
	repo = repoSplit[1]
	if commitSHA == "" {
		log.Fatalf("no commit SHA")
	}
	prNumber = 0
	if prNumberStr != "" {
		var err error
		if prNumber, err = strconv.Atoi(prNumberStr); err != nil {
			log.Fatalf("error encountered while parsing PR number: %s", err)
		}
	}

	if prNumber == 0 && branchName != "master" {
		log.Fatalf("no PR branch name supplied or push trigger not on master branch")
	}

	if err := postResult(validatorId, version); err != nil {
		log.Fatal(err)
	}
}
