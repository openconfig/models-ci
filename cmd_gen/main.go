package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openconfig/models-ci/commonci"
	"gopkg.in/yaml.v3"
)

var (
	// Commandline flags
	modelRoot          string // modelRoot is the root directory of the models.
	repoSlug           string // repoSlug is the "owner/repo" name of the models repo (e.g. openconfig/public).
	commitSHA          string
	prNumber           int
	extraPyangVersions string // e.g. "1.2.3,3.4.5"

	// Derived flags (for ease of use)
	owner string
	repo  string

	// local run flags
	local             bool   // local run toggle
	localResultsDir   string // folder into which the command outputs its results
	localValidatorId  string
	localModelDirName string // a model directory (e.g. network-instance, aft)

	// disabledModelPaths are the paths whose models should not undergo CI.
	// These should be temporary -- they're only here to help the transition to CI.
	// To represent a multi-level directory, use ":" instead of "/" as the delimiter.
	disabledModelPaths = map[string]bool{
		"wifi:access-points": true,
		"wifi:ap-manager":    true,
		"wifi:mac":           true,
		"wifi:phy":           true,
	}
)

func init() {
	// GCB-required flags
	flag.StringVar(&modelRoot, "modelRoot", "", "root directory to OpenConfig models")
	flag.StringVar(&repoSlug, "repo-slug", "openconfig/public", "repo where CI is run")
	flag.StringVar(&commitSHA, "commit-sha", "", "commit SHA of the PR")
	flag.IntVar(&prNumber, "pr-number", 0, "PR number")
	flag.StringVar(&extraPyangVersions, "extra-pyang-versions", "", "comma-separated extra pyang versions to run")

	// Local run flags
	flag.BoolVar(&local, "local", false, "use with validator, modelDirName, resultsDir to get a particular model's command")
	flag.StringVar(&localResultsDir, "resultsDir", "~/tmp/ci-results", "root directory to OpenConfig models")
	flag.StringVar(&localValidatorId, "validator", "", "")
	flag.StringVar(&localModelDirName, "modelDirName", "", "")
}

// ModelInfo represents the yaml model of an OpenConfig .spec.yml file.
type ModelInfo struct {
	Name       string
	DocFiles   []string `yaml:"docs"`
	BuildFiles []string `yaml:"build"`
	RunCi      bool     `yaml:"run-ci"`
}

// OpenConfigModelMap represents the directory structure and model information
// of the entire OpenConfig models required for CI.
type OpenConfigModelMap struct {
	// ModelRoot is the path to the OpenConfig models root directory.
	ModelRoot string
	// ModelInfoMap stores all ModelInfo for each model directory keyed by
	// the relative path to the model directory's .spec.yml.
	ModelInfoMap map[string][]ModelInfo
}

// parseModels walks the path given at modelRoot to populate the OpenConfigModelMap.
func parseModels(modelRoot string) (OpenConfigModelMap, error) {
	modelInfoMap := map[string][]ModelInfo{}
	err := filepath.Walk(modelRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
		}
		if !info.IsDir() && info.Name() == ".spec.yml" {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open spec file at path %q: %v\n", path, err)
			}
			m := []ModelInfo{}
			if err := yaml.NewDecoder(file).Decode(&m); err != nil {
				return fmt.Errorf("error while unmarshalling spec file at path %q: %v\n", path, err)
			}

			// Change the build paths to the absolute correct paths.
			for _, info := range m {
				for i, fileName := range info.BuildFiles {
					info.BuildFiles[i] = filepath.Join(modelRoot, strings.TrimPrefix(fileName, "yang/"))
				}
			}

			relPath, err := filepath.Rel(modelRoot, filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("failed to calculate relpath at path %q (modelRoot %q): %v\n", path, modelRoot, err)
			}
			// Allow nested model directories to be used later on as a partial file name.
			relPath = strings.ReplaceAll(relPath, "/", ":")
			modelInfoMap[relPath] = m
		}
		return nil
	})

	return OpenConfigModelMap{ModelRoot: modelRoot, ModelInfoMap: modelInfoMap}, err
}

// createValidatorFmtStr creates the customized format string to invoke the
// given validator on a model.
func createValidatorFmtStr(validatorId string) (string, error) {
	switch validatorId {
	case "pyang":
		return `if ! $@ -p %s -p %s/third_party/ietf %s &> %s; then
  mv %s %s
fi
`, nil
	case "oc-pyang":
		return `if ! $@ -p %s -p %s/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH %s &> %s; then
  mv %s %s
fi
`, nil
	case "pyangbind":
		return `if ! $@ -p %s -p %s/third_party/ietf -f pybind -o binding.py %s &> %s; then
  mv %s %s
fi
`, nil
	case "goyang-ygot":
		return `if ! go run /go/src/github.com/openconfig/ygot/generator/generator.go \
-path=%s,%s/third_party/ietf \
-output_file=/go/src/github.com/openconfig/ygot/exampleoc/oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
%s &> %s; then
  mv %s %s
fi
`, nil
	case "yanglint":
		return `if ! yanglint -p %s -p %s/third_party/ietf %s &> %s; then
  mv %s %s
fi
`, nil
	}
	return "", fmt.Errorf("createValidatorFmtStr: unrecognized validatorId %q", validatorId)
}

// genValidatorCommandForModelDir generates the validator command for a single modelDir.
func genValidatorCommandForModelDir(validatorId, resultsDir, modelDirName string, modelMap OpenConfigModelMap) (string, error) {
	var builder strings.Builder
	fmtStr, err := createValidatorFmtStr(validatorId)
	if err != nil {
		return "", err
	}
	for _, modelInfo := range modelMap.ModelInfoMap[modelDirName] {
		// First check whether to skip CI.
		if !modelInfo.RunCi || len(modelInfo.BuildFiles) == 0 {
			continue
		}
		outputFile := filepath.Join(resultsDir, fmt.Sprintf("%s==%s==pass", modelDirName, modelInfo.Name))
		failFile := filepath.Join(resultsDir, fmt.Sprintf("%s==%s==fail", modelDirName, modelInfo.Name))
		builder.WriteString(fmt.Sprintf(fmtStr, modelMap.ModelRoot, commonci.RootDir, strings.Join(modelInfo.BuildFiles, " "), outputFile, outputFile, failFile))
	}
	return builder.String(), nil
}

// labelPoster is an interface with just a function for posting a GitHub label to a PR.
type labelPoster interface {
	PostLabel(labelName, labelColor, owner, repo string, prNumber int) error
}

// genOpenConfigValidatorScript generates the whole validation script for the given validator.
// Tool version should be "" unless a non-head version is used.
// Scripts generated by this function assume the following:
//  1. Each validator uses a different command which can be customized, but all
//     will be run only on a single model as specified in the .spec.yml file.
//  2. Thus, a validation command and result is provided for each model.
//  2. A file indicating pass/fail is output for each model into the given result directory.
// Files names follow the "modelDir==model==status" format with no file extensions.
// The local flag indicates to run this as a helper to generate the script,
// rather than running it within GCB.
func genOpenConfigValidatorScript(g labelPoster, validatorId, version string, modelMap OpenConfigModelMap) (string, error) {
	resultsDir := commonci.ValidatorResultsDir(validatorId, version)
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("#!/bin/bash\nmkdir -p %s\n", resultsDir))

	modelDirNames := make([]string, 0, len(modelMap.ModelInfoMap))
	for modelDirName := range modelMap.ModelInfoMap {
		modelDirNames = append(modelDirNames, modelDirName)
	}
	sort.Strings(modelDirNames)

	for _, modelDirName := range modelDirNames {
		if disabledModelPaths[modelDirName] {
			log.Printf("skipping disabled model directory %s", modelDirName)
			g.PostLabel("skipped: "+modelDirName, commonci.LabelColors["orange"], owner, repo, prNumber)
			continue
		}
		cmdStr, err := genValidatorCommandForModelDir(validatorId, resultsDir, modelDirName, modelMap)
		if err != nil {
			return "", err
		}
		builder.WriteString(cmdStr)
	}

	return builder.String(), nil
}

// postInitialStatuses posts the initial status for all versions of a validator.
func postInitialStatuses(g *commonci.GithubRequestHandler, validatorId string, versions []string, prApproved bool) []error {
	var errs []error
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return append(errs, fmt.Errorf("validator %q not recognized", validatorId))
	}
	for _, version := range versions {
		validatorName := validator.Name + version
		// Update the status to pending so that the user can see that we have received
		// this request and are ready to run the CI.
		update := &commonci.GithubPRUpdate{
			Owner:       owner,
			Repo:        repo,
			Ref:         commitSHA,
			Description: validatorName + " Running",
			NewStatus:   "pending",
			Context:     validatorName,
		}
		if !prApproved && !validator.RunBeforeApproval {
			update.Description = validatorName + " Skipped (PR not approved)"
			update.NewStatus = "error"
		}

		if err := g.UpdatePRStatus(update); err != nil {
			log.Printf("error: couldn't update PR: %s", err)
			errs = append(errs, err)
		}
	}
	return errs
}

func main() {
	// Parse derived flags.
	flag.Parse()

	if modelRoot == "" {
		log.Fatalf("Must supply modelRoot path")
	}

	// Populate information necessary for validation script generation.
	modelMap, err := parseModels(modelRoot)
	if err != nil {
		log.Fatalf("CI flow failed due to error encountered while parsing spec files, parseModels: %v", err)
	}

	// Handle local call case.
	if local {
		if localModelDirName == "" {
			log.Fatalf("no modelDirName specified")
		}
		if localValidatorId == "" {
			log.Fatalf("no validator specified")
		}
		cmdStr, err := genValidatorCommandForModelDir(localValidatorId, localResultsDir, localModelDirName, modelMap)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(cmdStr)
		return
	} else if localModelDirName != "" || localValidatorId != "" {
		log.Fatalf("modelDirName and validator can only be specified for local cmd generation")
	}

	repoSplit := strings.Split(repoSlug, "/")
	owner = repoSplit[0]
	repo = repoSplit[1]
	if commitSHA == "" {
		log.Fatalf("no commit SHA")
	}
	if prNumber == 0 {
		log.Fatalf("no PR number")
	}

	h, err := commonci.NewGitHubRequestHandler()
	if err != nil {
		log.Fatal(err)
	}

	prApproved, err := h.IsPRApproved(owner, repo, prNumber)
	if err != nil {
		log.Fatalf("warning: Could not check PR approved status, running all checks: %v", err)
		prApproved = true
	}

	if err := os.MkdirAll(commonci.ResultsDir, 0644); err != nil {
		log.Fatalf("error while creating directory %q: %v", commonci.ResultsDir, err)
	}

	// Generate validation scripts, files, and post initial status on GitHub.
	for validatorId, validator := range commonci.Validators {
		// Empty string is the "head" version, which is always run.
		versionsToRun := []string{""}
		if validatorId == "pyang" {
			versionsToRun = append(versionsToRun, strings.Split(extraPyangVersions, ",")...)
		}
		// Write a list of the extra validator versions into the
		// designated extra versions file in order to be relayed to the
		// corresponding test.sh (next stage of the CI pipeline).
		extraVersionFile := filepath.Join(commonci.ResultsDir, fmt.Sprintf("extra-%s-versions.txt", validatorId))
		if err := ioutil.WriteFile(extraVersionFile, []byte(strings.Join(versionsToRun, " ")), 0444); err != nil {
			log.Fatalf("error while writing extra versions file %q: %v", extraVersionFile, err)
		}

		if errs := postInitialStatuses(h, validatorId, versionsToRun, prApproved); errs != nil {
			log.Fatal(errs)
		}
		if !prApproved && !validator.RunBeforeApproval {
			// We don't run less important and long tests until PR is approved.
			continue
		}
		// Generate validation commands for the validator.
		for _, version := range versionsToRun {
			validatorResultsDir := commonci.ValidatorResultsDir(validatorId, version)
			if err := os.MkdirAll(validatorResultsDir, 0644); err != nil {
				log.Fatalf("error while creating directory %q: %v", validatorResultsDir, err)
			}
			log.Printf("Created results directory %q", validatorResultsDir)

			scriptStr, err := genOpenConfigValidatorScript(h, validatorId, version, modelMap)
			if err != nil {
				log.Fatalf("error while generating validator script: %v", err)
			}
			scriptPath := filepath.Join(validatorResultsDir, commonci.ScriptFileName)
			if err := ioutil.WriteFile(scriptPath, []byte(scriptStr), 0744); err != nil {
				log.Fatalf("error while writing script to path %q: %v", scriptPath, err)
			}
		}
	}
}
