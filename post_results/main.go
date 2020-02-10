package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"log"

	"github.com/wenovus/models-ci/commonci"
)

const (
	// The title of the comment uses the relevant emoji to show whether it
	// succeeded or failed - so populate this based on the success of the test.
	mdPassSymbol       = ":white_check_mark:"
	mdFailSymbol       = ":no_entry:"
	IgnoreLintWarnings = true
)

var (
	// flags
	validatorId  string
	modelRoot    string
	repoSlug     string
	prBranchName string
	commitSHA    string
	version      string

	// derived flags
	owner     string
	repo      string
	validator *commonci.Validator
)

func init() {
	flag.StringVar(&validatorId, "validator", "", "unique name of the validator")
	flag.StringVar(&modelRoot, "modelRoot", "", "root directory to OpenConfig models")
	flag.StringVar(&repoSlug, "repo-slug", "openconfig/public", "repo where CI is run")
	flag.StringVar(&prBranchName, "pr-branch", "", "branch name of PR")
	flag.StringVar(&commitSHA, "commit-sha", "", "commit SHA of the PR")
	flag.StringVar(&version, "version", "", "version of the validator tool")
}

func lintSymbol(pass bool) string {
	if !pass {
		return mdFailSymbol
	}
	return mdPassSymbol
}

func processAnyPyangOutput(rawOut string, pass, noWarnings bool) (string, error) {
	var errorLines, nonErrorLines strings.Builder
	for _, line := range strings.Split(rawOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sections := strings.SplitN(line, ":", 4)
		if len(sections) < 4 {
			nonErrorLines.WriteString("  <li>")
			nonErrorLines.WriteString(line)
			nonErrorLines.WriteString("</li>\n")
			continue
		}

		filePath := strings.TrimSpace(sections[0])
		var err error
		filePath, err = filepath.Rel(modelRoot, filePath)
		if err != nil {
			return "", fmt.Errorf("failed to calculate relpath at path %q (modelRoot %q) parsed from message %q: %v\n", filePath, modelRoot, line, err)
		}
		lineNumber := strings.TrimSpace(sections[1])
		errorLevel := strings.TrimSpace(sections[2])
		message := strings.TrimSpace(sections[3])

		// Get rid of subpath information with the line number as this is not useful to users.
		subpathIndex := strings.Index(lineNumber, "(")
		if subpathIndex != -1 {
			messageSections := strings.SplitN(message, ":", 2)
			if len(messageSections) == 1 {
				// When there is subpath information, we expect there to be an extra colon due to the
				// subpath line number; so, this is unrecognized format.
				nonErrorLines.WriteString("  <li>")
				nonErrorLines.WriteString(line)
				nonErrorLines.WriteString("</li>\n")
				continue
			}
			lineNumber = strings.TrimSpace(lineNumber[:subpathIndex])
			errorLevel = strings.TrimSpace(messageSections[0])
			message = strings.TrimSpace(messageSections[1])
		}

		writtenLine := fmt.Sprintf("  <li>%s (%s): %s: <pre>%s</pre></li>\n", filePath, lineNumber, errorLevel, message)
		if strings.Contains(strings.ToLower(errorLevel), "error") {
			errorLines.WriteString(writtenLine)
		} else if noWarnings && strings.Contains(strings.ToLower(errorLevel), "warning") {
			continue
		} else {
			nonErrorLines.WriteString(writtenLine)
		}
	}

	var out strings.Builder
	if pass {
		out.WriteString("Passed.\n")
	}
	errorOut := errorLines.String()
	nonErrorOut := nonErrorLines.String()
	if errorOut != "" || nonErrorOut != "" {
		out.WriteString("<ul>\n")
		out.WriteString(errorLines.String())
		out.WriteString(nonErrorLines.String())
		out.WriteString("</ul>\n")
	}
	return out.String(), nil
}

// parseResultsMd transforms the output files of the validator script into MD
// to be displayed on GitHub.
func parseResultsMd(validatorResultDir string) (string, bool, error) {
	var md, modelMd strings.Builder
	firstModelDir := true
	var prevModelDirName string
	allPass := true
	modelDirPass := true
	if err := filepath.Walk(validatorResultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
		}

		components := strings.Split(info.Name(), "==")
		// Handle per-model output
		if !info.IsDir() && len(components) == 3 {
			outBytes, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file at path %q: %v\n", path, err)
			}

			modelDirName := components[0]
			modelName := components[1]

			// Whenever a new modelDir finishes processing, then write the whole thing; this is instead of a model at a time.
			// This is because the status for the model dir depends on the status of every single model it contains.
			if prevModelDirName == "" {
				prevModelDirName = modelDirName
			} else if modelDirName != prevModelDirName {
				if firstModelDir {
					md.WriteString(fmt.Sprintf("<details>\n  <summary>%s %s</summary>\n", lintSymbol(modelDirPass), prevModelDirName))
					firstModelDir = false
				} else {
					md.WriteString(fmt.Sprintf("</details>\n<details>\n  <summary>%s %s</summary>\n", lintSymbol(modelDirPass), prevModelDirName))
				}
				md.WriteString(modelMd.String())
				modelMd.Reset()
				prevModelDirName = modelDirName
				modelDirPass = true
			}

			// This has to go after modelDirPass's reset to "true"
			// in order to correctly set modelDirPass with the
			// result of the first model in the directory.
			var modelPass bool
			var outString string
			switch components[2] {
			case "pass":
				modelPass = true
				outString = "Passed.\n"
			case "fail":
				modelPass = false
				outString = "Failed.\n"
				allPass = false
				modelDirPass = false
			default:
				return fmt.Errorf("expect status at path %q to be true or false, got %v", path, components[2])
			}

			if strings.Contains(validatorId, "pyang") {
				outString, err = processAnyPyangOutput(string(outBytes), modelPass, IgnoreLintWarnings)
			} else if outStr := string(outBytes); outStr != "" {
				outStr = strings.Join(strings.Split(outStr, "\n"), "<br>\n")
				if modelPass {
					outString += outStr
				} else {
					outString = outStr
				}
			}

			if err != nil {
				return fmt.Errorf("error encountered while processing output: %v", err)
			}

			modelMd.WriteString(fmt.Sprintf("<details>\n  <summary>%s %s</summary>\n  %s</details>\n",
				lintSymbol(modelPass), modelName, outString))
		}
		return nil
	}); err != nil {
		return "", false, err
	}

	if firstModelDir {
		md.WriteString(fmt.Sprintf("<details>\n  <summary>%s %s</summary>\n", lintSymbol(modelDirPass), prevModelDirName))
	} else {
		md.WriteString(fmt.Sprintf("</details>\n<details>\n  <summary>%s %s</summary>\n", lintSymbol(modelDirPass), prevModelDirName))
	}
	md.WriteString(modelMd.String())
	// Close the last model directory.
	md.WriteString("</details>\n")

	return md.String(), allPass, nil
}

// postResult runs the OpenConfig linter, and Go-based tests for the models
// repo. The results are written to a GitHub Gist, and into the PR that was
// modified, associated with the commit reference SHA.
func postResult() {
	var url, gistID string
	var err error
	var g *commonci.GithubRequestHandler
	commonci.Retry(5, "CreateCIOutputGist", func() error {
		g = commonci.NewGitHubRequestHandler()
		url, gistID, err = g.CreateCIOutputGist(validatorId, version)
		return err
	})
	if err != nil {
		log.Fatalf("error: couldn't create gist: %v", err)
	}

	var outString string
	pass := false
	scriptFailPath := filepath.Join(commonci.ResultsDir, validatorId, commonci.FailFileName)
	failFileBytes, err := ioutil.ReadFile(scriptFailPath)
	// A non-existent or an empty fail file is a pass.
	var failOutString string
	if err == nil {
		failOutString = string(failFileBytes)
	}

	switch {
	case failOutString != "":
		outString = failOutString
	case !commonci.Validators[validatorId].IsPerModel:
		outString = "Test passed"
		pass = true
	default:
		outString, pass, err = parseResultsMd(filepath.Join(commonci.ResultsDir, validatorId))
		if err != nil {
			log.Fatalf("error, couldn't parse results: %v", err)
		}
	}

	validatorName := validator.Name + version

	g.AddGistComment(gistID, outString, fmt.Sprintf("%s %s", lintSymbol(pass), validatorName))

	prUpdate := &commonci.GithubPRUpdate{
		Owner:   owner,
		Repo:    repo,
		Ref:     commitSHA,
		URL:     url,
		Context: validatorName,
	}

	if !pass {
		prUpdate.NewStatus = "failure"
		prUpdate.Description = validatorName + " Failed"

		if uperr := g.UpdatePRStatus(prUpdate); uperr != nil {
			log.Printf("error: couldn't update PR to failed, error: %s", uperr)
		}
		return
	}

	prUpdate.NewStatus = "success"
	prUpdate.Description = validatorName + " Succeeded"
	if uperr := g.UpdatePRStatus(prUpdate); uperr != nil {
		log.Printf("error: couldn't update PR to succeeded: %s", uperr)
	}
}

func main() {
	flag.Parse()
	repoSplit := strings.Split(repoSlug, "/")
	owner = repoSplit[0]
	repo = repoSplit[1]
	if commitSHA == "" {
		log.Fatalf("no commit SHA")
	}
	if prBranchName == "" {
		log.Fatalf("no PR branch name supplied")
	}

	validator = commonci.Validators[validatorId]
	if validator == nil {
		log.Fatalf("validator %q not found!", validatorId)
	}

	postResult()
}
