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
	"strings"

	"log"

	"github.com/openconfig/models-ci/commonci"
)

// post_results posts the CI results for a given tool using the output from
// running the tool's script generated by cmd_gen. The location of the results
// is determined by common_ci.

const (
	// The title of the results uses the relevant emoji to show whether it
	// succeeded or failed.
	mdPassSymbol    = ":white_check_mark:"
	mdWarningSymbol = ":warning:"
	mdFailSymbol    = ":no_entry:"
	// IgnorePyangWarnings ignores all warnings from pyang or pyang-based tools.
	IgnorePyangWarnings = false
	// IgnoreConfdWarnings ignores all warnings from ConfD.
	IgnoreConfdWarnings = false
)

var (
	// flags
	validatorId  string // validatorId is the unique name identifying the validator (see commonci for all of them)
	modelRoot    string // modelRoot is the root directory of the models.
	repoSlug     string // repoSlug is the "owner/repo" name of the models repo (e.g. openconfig/public).
	prNumber     int
	prBranchName string
	commitSHA    string
	version      string // version is a specific version of the validator that's being run (empty means latest).

	// derived flags
	owner string
	repo  string
)

func init() {
	flag.StringVar(&validatorId, "validator", "", "unique name of the validator")
	flag.StringVar(&modelRoot, "modelRoot", "", "root directory to OpenConfig models")
	flag.StringVar(&repoSlug, "repo-slug", "", "repo where CI is run")
	flag.IntVar(&prNumber, "pr-number", 0, "PR number")
	flag.StringVar(&prBranchName, "pr-branch", "", "branch name of PR")
	flag.StringVar(&commitSHA, "commit-sha", "", "commit SHA of the PR")
	flag.StringVar(&version, "version", "", "(optional) specific version of the validator tool.")
}

type CheckStatus int

const (
	Pass CheckStatus = iota
	Warning
	Fail
)

func (c CheckStatus) String() string {
	return [...]string{"Pass", "Warning", "Fail"}[c]
}

func lintSymbol(status CheckStatus) string {
	switch status {
	case Pass:
		return mdPassSymbol
	case Warning:
		return mdWarningSymbol
	}
	return mdFailSymbol
}

// sprintLineHTML prints a single list item to be put under a top-level summary item.
func sprintLineHTML(format string, a ...interface{}) string {
	return fmt.Sprintf("  <li>"+format+"</li>\n", a...)
}

// sprintSummaryHTML prints a top-level summary item containing free-form or list items.
func sprintSummaryHTML(status CheckStatus, title, format string, a ...interface{}) string {
	return fmt.Sprintf("<details>\n  <summary>%s %s</summary>\n"+format+"</details>\n", append([]interface{}{lintSymbol(status), title}, a...)...)
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
			case "openconfig-version":
				fallthrough
			case "latest-revision-version":
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

// processMiscChecksOutput takes the raw result output from the misc-checks
// results directory and returns its formatted report and status.
func processMiscChecksOutput(resultsDir string) (string, CheckStatus, error) {
	fileProperties := map[string]map[string]string{}
	changedFiles, err := readYangFilesList(filepath.Join(resultsDir, "changed-files.txt"))
	if err != nil {
		return "", Fail, err
	}
	for _, file := range changedFiles {
		if _, ok := fileProperties[file]; !ok {
			fileProperties[file] = map[string]string{}
		}
		fileProperties[file]["changed"] = "true"
	}
	if err := readGoyangVersionsLog(filepath.Join(resultsDir, "pr-file-parse-log"), false, fileProperties); err != nil {
		return "", Fail, err
	}
	if err := readGoyangVersionsLog(filepath.Join(resultsDir, "master-file-parse-log"), true, fileProperties); err != nil {
		return "", Fail, err
	}

	var ocVersionViolations []string
	ocVersionChangedCount := 0
	var reachabilityViolations []string
	filesReachedCount := 0
	// Only look at the PR's files as they might be different from the master's files.
	allNonEmptyPRFiles, err := readYangFilesList(filepath.Join(resultsDir, "all-non-empty-files.txt"))
	if err != nil {
		return "", Fail, err
	}
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
		case hasVersion:
			// TODO(wenovus): This logic can be improved to check whether the increment follows semver rules.
			if ocVersion == masterOcVersion {
				ocVersionViolations = append(ocVersionViolations, sprintLineHTML("%s: file updated but PR version not updated: %q", file, ocVersion))
				break
			}
			ocVersionChangedCount += 1
		case hadVersion:
			ocVersionViolations = append(ocVersionViolations, sprintLineHTML("%s: openconfig-version was removed", file))
		}
	}

	// Compute HTML string and status.
	var out strings.Builder
	status := Pass
	appendViolationOut := func(desc string, violations []string, passString string) {
		if len(violations) == 0 {
			out.WriteString(sprintSummaryHTML(Pass, desc, passString))
		} else {
			out.WriteString(sprintSummaryHTML(Fail, desc, strings.Join(violations, "")))
			status = Fail
		}
	}
	appendViolationOut("openconfig-version update check", ocVersionViolations, fmt.Sprintf("%d file(s) correctly updated.\n", ocVersionChangedCount))
	appendViolationOut(".spec.yml build reachability check", reachabilityViolations, fmt.Sprintf("%d files reached by build rules.\n", filesReachedCount))

	return out.String(), status, nil
}

// processStandardOutput takes raw pyang/confd output and transforms it to an
// HTML format for display on a GitHub gist comment.
// Both types of validators output a string following the format:
// <file path>:<line no>:<error/warning>:<message>
// pyang also has a second format:
// <file path>:<line#>(<sub file path>:<line#>):<error/warning>:<message>
// Errors are displayed in front of warnings.
// The bool return value indicates whether there is any warning in the output.
func processStandardOutput(rawOut string, pass, noWarnings bool) (string, bool, error) {
	var errorLines, warningLines, unrecognizedLines strings.Builder
	for _, line := range strings.Split(rawOut, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		sections := strings.SplitN(line, ":", 4)
		// warning/error lines from pyang/confd have a "path:line#:status:message" format.
		if len(sections) < 4 {
			unrecognizedLines.WriteString(sprintLineHTML(line))
			continue
		}
		filePath := strings.TrimSpace(sections[0])
		lineNumber := strings.TrimSpace(sections[1])
		status := strings.ToLower(strings.TrimSpace(sections[2]))
		message := strings.TrimSpace(sections[3])

		// Convert file path to relative path.
		var err error
		if filePath, err = filepath.Rel(modelRoot, filePath); err != nil {
			return "", false, fmt.Errorf("failed to calculate relpath at path %q (modelRoot %q) parsed from message %q: %v\n", filePath, modelRoot, line, err)
		}

		// When there is subpath information, remove it (as it's not useful to users) and re-compute information.
		// path:line#(subpath:line#):status:message
		subpathIndex := strings.Index(sections[1], "(")
		if subpathIndex != -1 {
			messageSections := strings.SplitN(sections[3], ":", 2)
			if len(messageSections) == 1 {
				// When there is subpath information, we expect there to be an extra colon due to the
				// subpath line number; so, this is unrecognized format.
				unrecognizedLines.WriteString(sprintLineHTML(line))
				continue
			}
			lineNumber = strings.TrimSpace(sections[1][:subpathIndex])
			status = strings.ToLower(strings.TrimSpace(messageSections[0]))
			message = strings.TrimSpace(messageSections[1])
		}

		processedLine := fmt.Sprintf("%s (%s): %s: <pre>%s</pre>", filePath, lineNumber, status, message)
		switch {
		case strings.Contains(status, "error"):
			errorLines.WriteString(sprintLineHTML(processedLine))
		case strings.Contains(status, "warning"):
			if !noWarnings {
				warningLines.WriteString(sprintLineHTML(processedLine))
			}
		default: // Unrecognized line, so write unprocessed output.
			unrecognizedLines.WriteString(sprintLineHTML(line))
		}
	}

	var out strings.Builder
	if pass {
		out.WriteString("Passed.\n")
	}
	if errorLines.Len() > 0 || warningLines.Len() > 0 || unrecognizedLines.Len() > 0 {
		out.WriteString("<ul>\n")
		out.WriteString(errorLines.String())
		out.WriteString(warningLines.String())
		out.WriteString(unrecognizedLines.String())
		out.WriteString("</ul>\n")
	}
	return out.String(), warningLines.Len() > 0, nil
}

// parseModelResultsHTML transforms the output files of the validator script into HTML
// to be displayed on GitHub.
func parseModelResultsHTML(validatorId, validatorResultDir string) (string, CheckStatus, error) {
	var htmlOut, modelHTML strings.Builder
	var prevModelDirName string

	overallStatus := Pass
	modelDirStatus := Pass
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
				htmlOut.WriteString(sprintSummaryHTML(modelDirStatus, prevModelDirName, modelHTML.String()))
				modelHTML.Reset()
				modelDirStatus = Pass
			}
			prevModelDirName = modelDirName

			// modelPass can only be either pass or fail from the validator's execution return code.
			modelPass := true
			switch status {
			case "pass":
			case "fail":
				overallStatus = Fail
				modelDirStatus = Fail
				modelPass = false
			default:
				return fmt.Errorf("expect status at path %q to be true or false, got %v", path, status)
			}

			// Get output string.
			outString, err := readFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file at path %q: %v\n", path, err)
			}

			// Transform output string into HTML.
			var warning bool
			switch {
			case strings.Contains(validatorId, "pyang"):
				outString, warning, err = processStandardOutput(outString, modelPass, IgnorePyangWarnings)
			case validatorId == "confd":
				outString, warning, err = processStandardOutput(outString, modelPass, IgnoreConfdWarnings)
			default:
				outString = strings.Join(strings.Split(outString, "\n"), "<br>\n")
				if modelPass {
					outString = "Passed.\n" + outString
				}
			}
			if !modelPass && outString == "" {
				outString = "Failed.\n"
			}
			if warning {
				if overallStatus != Fail {
					overallStatus = Warning
				}
				if modelDirStatus != Fail {
					// FIXME(wenovus): Add tests.
					modelDirStatus = Warning
				}
			}
			if err != nil {
				return fmt.Errorf("error encountered while processing output for validator %q: %v", validatorId, err)
			}

			if modelPass {
				modelHTML.WriteString(sprintSummaryHTML(Pass, modelName, outString))
			} else {
				modelHTML.WriteString(sprintSummaryHTML(Fail, modelName, outString))
			}
		}
		return nil
	}); err != nil {
		return "", Fail, err
	}

	// Edge case: handle last modelDir.
	htmlOut.WriteString(sprintSummaryHTML(modelDirStatus, prevModelDirName, modelHTML.String()))

	return htmlOut.String(), overallStatus, nil
}

// getResult parses the results for the given validator and its results
// directory, and returns the string to be put in a GitHub gist comment as well
// as the status.
func getResult(validatorId, resultsDir string) (string, CheckStatus, error) {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return "", Fail, fmt.Errorf("validator %q not found!", validatorId)
	}

	// outString is parsed stdout.
	var outString string
	// status is the overall validation result.
	var status CheckStatus

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
		status = Fail
	case validator.IsPerModel && validatorId == "misc-checks":
		outString, status, err = processMiscChecksOutput(resultsDir)
	case validator.IsPerModel:
		outString, status, err = parseModelResultsHTML(validatorId, resultsDir)
	default:
		outString = "Test passed."
		status = Pass
	}

	return outString, status, err
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
			log.Printf("did not read latest version for %s: %v", validatorId, err)
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
		testResultString, status, err := getResult(vv.ValidatorId, resultsDir)
		if err != nil {
			return fmt.Errorf("postResult: couldn't parse results for <%s>@<%s> in resultsDir %q: %v", vv.ValidatorId, vv.Version, resultsDir, err)
		}

		gistTitle := fmt.Sprintf("%s %s", lintSymbol(status), validatorDescs[i])
		gistContent := testResultString
		id, err := g.AddGistComment(gistID, gistTitle, gistContent)
		if err != nil {
			fmt.Errorf("postResult: could not add gist comment: %v", err)
		}

		commentBuilder.WriteString(fmt.Sprintf("%s [%s](%s#gistcomment-%d)\n", lintSymbol(status), validatorDescs[i], gistURL, id))
	}
	comment := commentBuilder.String()
	if err := g.AddPRComment(&comment, owner, repo, prNumber); err != nil {
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

	var url, gistID string
	var err error
	var g *commonci.GithubRequestHandler

	compatReportsStr, err := readFile(commonci.CompatReportValidatorsFile)
	if err != nil {
		return fmt.Errorf("postResult: %v", err)
	}
	compatValidators, compatValidatorsMap := commonci.GetCompatReportValidators(compatReportsStr)

	if validatorId == "compat-report" {
		log.Printf("Processing compatibility report for %s", compatReportsStr)
		return postCompatibilityReport(compatValidators)
	}

	// Skip reporting if validator is part of compatibility report.
	if compatValidatorsMap[validatorId][version] {
		log.Printf("Validator %s part of compatibility report, skipping reporting standalone PR status.", commonci.AppendVersionToName(validatorId, version))
		return nil
	}
	resultsDir := commonci.ValidatorResultsDir(validatorId, version)

	// Create gist representing test results. The "validatorDesc" is the
	// title of the gist, and "content" is the script execution output.
	validatorDesc, content, err := getGistHeading(validatorId, version, resultsDir)
	if err != nil {
		return fmt.Errorf("postResult: %v", err)
	}
	if err := commonci.Retry(5, "CreateCIOutputGist", func() error {
		g, err = commonci.NewGitHubRequestHandler()
		if err != nil {
			return err
		}
		url, gistID, err = g.CreateCIOutputGist(validatorDesc, content)
		return err
	}); err != nil {
		return fmt.Errorf("postResult: couldn't create gist: %v", err)
	}

	// Post parsed test results as a gist comment.
	testResultString, status, err := getResult(validatorId, resultsDir)
	if err != nil {
		return fmt.Errorf("postResult: couldn't parse results: %v", err)
	}
	if _, err := g.AddGistComment(gistID, fmt.Sprintf("%s %s", lintSymbol(status), validatorDesc), testResultString); err != nil {
		fmt.Errorf("postResult: could not add gist comment: %v", err)
	}

	prUpdate := &commonci.GithubPRUpdate{
		Owner:   owner,
		Repo:    repo,
		Ref:     commitSHA,
		URL:     url,
		Context: validator.StatusName(version),
	}
	switch status {
	case Pass:
		prUpdate.NewStatus = "success"
		prUpdate.Description = validatorDesc + " Succeeded"
	case Warning:
		prUpdate.NewStatus = "success"
		prUpdate.Description = validatorDesc + " Succeeded (warnings)"
	case Fail:
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
	if prBranchName == "" {
		log.Fatalf("no PR branch name supplied")
	}

	if err := postResult(validatorId, version); err != nil {
		log.Fatal(err)
	}
}
