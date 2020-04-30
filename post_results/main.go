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
	mdPassSymbol = ":white_check_mark:"
	mdFailSymbol = ":no_entry:"
	// IgnorePyangWarnings ignores all warnings from pyang or pyang-based tools.
	IgnorePyangWarnings = true
)

var (
	// flags
	validatorId  string // validatorId is the unique name identifying the validator (see commonci for all of them)
	modelRoot    string // modelRoot is the root directory of the models.
	repoSlug     string // repoSlug is the "owner/repo" name of the models repo (e.g. openconfig/public).
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
	flag.StringVar(&prBranchName, "pr-branch", "", "branch name of PR")
	flag.StringVar(&commitSHA, "commit-sha", "", "commit SHA of the PR")
	flag.StringVar(&version, "version", "", "(optional) specific version of the validator tool.")
}

func lintSymbol(pass bool) string {
	if !pass {
		return mdFailSymbol
	}
	return mdPassSymbol
}

func sprintLineHTML(format string, a ...interface{}) string {
	return fmt.Sprintf("  <li>"+format+"</li>\n", a...)
}

func sprintSummaryHTML(pass bool, title, message string) string {
	return fmt.Sprintf("<details>\n  <summary>%s %s</summary>\n%s</details>\n", lintSymbol(pass), title, message)
}

func readFile(path string) (string, error) {
	outBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file at path %q: %v\n", path, err)
	}
	return string(outBytes), nil
}

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

// TODO(wenovus): need comprehensive test cases.
func readGoyangVersionsLog(path string, masterBranch bool, fileProperties map[string]map[string]string) error {
	fileLog, err := readFile(path)
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
			return fmt.Errorf("while parsing %s: unrecognized line heading %q, expected a \"<name>.yang:\" start to the line: %q", path, yangFileName, line)
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
				return fmt.Errorf("while parsing %s: unrecognized property substring, expected \"<property name>:\"<property>\"\" separated by spaces: %q", path, property)
			}
			name, value := segments[0], segments[1]
			if value[0] == '"' {
				if len(value) == 1 || value[len(value)-1] != '"' {
					return fmt.Errorf("while parsing %s: Got invalid property value format: %s -- if the property value starts with a quote, it is assumed to be an enclosing quote", path, property)
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

func processMiscChecksOutput(testPath string) (string, bool, error) {
	fileProperties := map[string]map[string]string{}
	changedFiles, err := readYangFilesList(filepath.Join(testPath, "changed-files.txt"))
	if err != nil {
		return "", false, err
	}
	for _, file := range changedFiles {
		if _, ok := fileProperties[file]; !ok {
			fileProperties[file] = map[string]string{}
		}
		fileProperties[file]["changed"] = "true"
	}
	if err := readGoyangVersionsLog(filepath.Join(testPath, "pr-file-parse-log"), false, fileProperties); err != nil {
		return "", false, err
	}
	if err := readGoyangVersionsLog(filepath.Join(testPath, "master-file-parse-log"), true, fileProperties); err != nil {
		return "", false, err
	}

	var ocVersionViolations []string
	var reachabilityViolations []string
	// Only look at the PR's files as they might be different from the master's files.
	allNonEmptyPRFiles, err := readYangFilesList(filepath.Join(testPath, "all-non-empty-files.txt"))
	if err != nil {
		return "", false, err
	}
	for _, file := range allNonEmptyPRFiles {
		properties, ok := fileProperties[file]

		// Reachability check
		if !ok || properties["reachable"] != "true" {
			reachabilityViolations = append(reachabilityViolations, sprintLineHTML("%s: Non-null schema not used by any .spec.yml tree.", file))
			// If the file was not reached, then its other
			// parameters would not have been parsed by goyang, so
			// simply skip the rest of the checks.
			continue
		}

		// openconfig-version update check
		ocVersion := properties["openconfig-version"]
		if ocVersion == "" { // TODO(wenovus): need test case
			ocVersionViolations = append(ocVersionViolations, sprintLineHTML("%s: openconfig-version not found", file))
		} else if properties["changed"] == "true" {
			// TODO(wenovus): This logic can be improved to check whether the increment follows semver rules.
			if ocVersion == properties["master-openconfig-version"] {
				ocVersionViolations = append(ocVersionViolations, sprintLineHTML("%s: file updated but PR version not updated: %q", file, ocVersion))
			}
		}
	}

	// Compute HTML string and pass/fail status.
	var out strings.Builder
	var pass = true
	appendViolationOut := func(desc string, violations []string) {
		if len(violations) == 0 {
			out.WriteString(sprintSummaryHTML(true, desc, "Passed.\n"))
		} else {
			out.WriteString(sprintSummaryHTML(false, desc, strings.Join(violations, "")))
			pass = false
		}
	}
	appendViolationOut("openconfig-version update check", ocVersionViolations)
	appendViolationOut(".spec.yml build reachability check", reachabilityViolations)

	return out.String(), pass, nil
}

// processAnyPyangOutput takes the raw pyang output and transforms it to an
// HTML format for display on a GitHub gist comment.
func processAnyPyangOutput(rawOut string, pass, noWarnings bool) (string, error) {
	var errorLines, nonErrorLines strings.Builder
	for _, line := range strings.Split(rawOut, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		sections := strings.SplitN(line, ":", 4)
		// warning/error lines from pyang have a "path:line#:status:message" format.
		if len(sections) < 4 {
			nonErrorLines.WriteString(sprintLineHTML(line))
			continue
		}
		filePath := strings.TrimSpace(sections[0])
		lineNumber := strings.TrimSpace(sections[1])
		status := strings.ToLower(strings.TrimSpace(sections[2]))
		message := strings.TrimSpace(sections[3])

		// Convert file path to relative path.
		var err error
		if filePath, err = filepath.Rel(modelRoot, filePath); err != nil {
			return "", fmt.Errorf("failed to calculate relpath at path %q (modelRoot %q) parsed from message %q: %v\n", filePath, modelRoot, line, err)
		}

		// When there is subpath information, remove it (as it's not useful to users) and re-compute information.
		// path:line#(subpath:line#):status:message
		subpathIndex := strings.Index(sections[1], "(")
		if subpathIndex != -1 {
			messageSections := strings.SplitN(sections[3], ":", 2)
			if len(messageSections) == 1 {
				// When there is subpath information, we expect there to be an extra colon due to the
				// subpath line number; so, this is unrecognized format.
				nonErrorLines.WriteString(sprintLineHTML(line))
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
				nonErrorLines.WriteString(sprintLineHTML(processedLine))
			}
		default: // Unrecognized line, so write unprocessed output.
			nonErrorLines.WriteString(sprintLineHTML(line))
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

// parseModelResultsHTML transforms the output files of the validator script into HTML
// to be displayed on GitHub.
func parseModelResultsHTML(validatorId, validatorResultDir string) (string, bool, error) {
	var htmlOut, modelHTML strings.Builder
	var prevModelDirName string

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
				htmlOut.WriteString(sprintSummaryHTML(modelDirPass, prevModelDirName, modelHTML.String()))
				modelHTML.Reset()
				modelDirPass = true
			}
			prevModelDirName = modelDirName

			modelPass := true
			switch status {
			case "pass":
			case "fail":
				allPass = false
				modelDirPass = false
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
			if strings.Contains(validatorId, "pyang") {
				outString, err = processAnyPyangOutput(outString, modelPass, IgnorePyangWarnings)
				if err != nil {
					return fmt.Errorf("error encountered while processing output for validator %q: %v", validatorId, err)
				}
			} else {
				outString = strings.Join(strings.Split(outString, "\n"), "<br>\n")
				if modelPass {
					outString = "Passed.\n" + outString
				}
			}
			if !modelPass && outString == "" {
				outString = "Failed.\n"
			}

			modelHTML.WriteString(sprintSummaryHTML(modelPass, modelName, outString))
		}
		return nil
	}); err != nil {
		return "", false, err
	}

	// Edge case: handle last modelDir.
	htmlOut.WriteString(sprintSummaryHTML(modelDirPass, prevModelDirName, modelHTML.String()))

	return htmlOut.String(), allPass, nil
}

// getResult parses the results for the given validator and its results
// directory, and returns the string to be put in a GitHub gist comment as well
// as the status (i.e. pass or fail).
func getResult(validatorId, resultsDir string) (string, bool, error) {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return "", false, fmt.Errorf("validator %q not found!", validatorId)
	}

	// outString is parsed stdout.
	var outString string
	// pass is the overall validation result.
	var pass bool

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
	case !validator.IsPerModel:
		outString = "Test passed."
		pass = true
	default: // validator.IsPerModel
		outString, pass, err = parseModelResultsHTML(validatorId, resultsDir)
	}

	return outString, pass, err
}

// getGistInfo gets the description and content of the result gist for the
// given validator from its script output file. The "description" is the title
// of the gist, and "content" is the script execution output.
// NOTE: The parsed test result output (distinct from the script execution
// output) should be attached as a comment on the same gist.
func getGistInfo(validatorId, version, resultsDir string) (string, string, error) {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return "", "", fmt.Errorf("getGistInfo: validator %q not found!", validatorId)
	}

	validatorDesc := validator.StatusName(version)
	// If version is latest, then get the concrete version output by the tool if it exists.
	if version == "" {
		if outBytes, err := ioutil.ReadFile(filepath.Join(resultsDir, commonci.LatestVersionFileName)); err != nil {
			log.Printf("did not read latest version for %s: %v", validatorId, err)
		} else {
			// Get the first line of the version output as the tool's display title, with extra spacing in between words removed.
			validatorDesc = strings.Join(strings.Fields(strings.TrimSpace(strings.SplitN(string(outBytes), "\n", 2)[0])), " ")
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

// postResult runs the OpenConfig linter, and Go-based tests for the models
// repo. The results are written to a GitHub Gist, and into the PR that was
// modified, associated with the commit reference SHA.
func postResult(validatorId, version string) error {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return fmt.Errorf("postResult: validator %q not found!", validatorId)
	}

	var url, gistID string
	var err error
	var g *commonci.GithubRequestHandler

	resultsDir := commonci.ValidatorResultsDir(validatorId, version)

	// Create gist representing test results. The "validatorDesc" is the
	// title of the gist, and "content" is the script execution output.
	validatorDesc, content, err := getGistInfo(validatorId, version, resultsDir)
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
	testResultString, pass, err := getResult(validatorId, resultsDir)
	if err != nil {
		return fmt.Errorf("postResult: couldn't parse results: %v", err)
	}
	g.AddGistComment(gistID, fmt.Sprintf("%s %s", lintSymbol(pass), validatorDesc), testResultString)

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
	if prBranchName == "" {
		log.Fatalf("no PR branch name supplied")
	}

	if err := postResult(validatorId, version); err != nil {
		log.Fatal(err)
	}
}
