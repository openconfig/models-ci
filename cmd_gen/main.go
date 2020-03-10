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
	// flags
	modelRoot string
	repoSlug  string
	commitSHA string
	prNumber  int
	extraPV   string

	// derived flags
	owner string
	repo  string

	// disabledModelPaths are the paths whose models should not undergo CI.
	// These should be temporary.
	disabledModelPaths = map[string]bool{
		"wifi:access-points": true,
		"wifi:ap-manager":    true,
		"wifi:mac":           true,
		"wifi:phy":           true,
	}
)

func init() {
	flag.StringVar(&modelRoot, "modelRoot", "", "root directory to OpenConfig models")
	flag.StringVar(&repoSlug, "repo-slug", "openconfig/public", "repo where CI is run")
	flag.StringVar(&commitSHA, "commit-sha", "", "commit SHA of the PR")
	flag.IntVar(&prNumber, "pr-number", 0, "PR number")
	flag.StringVar(&extraPV, "extra-pyang-versions", "", "Extra pyang versions to run")
}

type ModelInfo struct {
	Name       string
	DocFiles   []string `yaml:"docs"`
	BuildFiles []string `yaml:"build"`
	RunCi      bool     `yaml:"run-ci"`
}

type OpenConfigModelMap struct {
	ModelRoot    string
	ModelInfoMap map[string][]ModelInfo
}

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

			parentPath := filepath.Dir(path)
			relPath, err := filepath.Rel(modelRoot, parentPath)
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

func createFmtStr(validatorId string) string {
	switch validatorId {
	case "pyang":
		return `if ! $@ -p %s -p %s/third_party/ietf %s &> %s; then
  mv %s %s
fi
`
	case "oc-pyang":
		return `if ! $@ -p %s -p %s/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH %s &> %s; then
  mv %s %s
fi
`
	case "pyangbind":
		return `if ! $@ -p %s -p %s/third_party/ietf -f pybind -o binding.py %s &> %s; then
  mv %s %s
fi
`
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
`
	case "yanglint":
		return `if ! yanglint -p %s -p %s/third_party/ietf %s &> %s; then
  mv %s %s
fi
`
	}
	return ""
}

func genOpenConfigLinterCmd(g *commonci.GithubRequestHandler, validatorId, folderPath string, modelMap OpenConfigModelMap) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("#!/bin/bash\nmkdir -p %s\n", folderPath))

	modelDirNames := make([]string, 0, len(modelMap.ModelInfoMap))
	for modelDirName := range modelMap.ModelInfoMap {
		modelDirNames = append(modelDirNames, modelDirName)
	}
	sort.Strings(modelDirNames)

	fmtStr := createFmtStr(validatorId)

	for _, modelDirName := range modelDirNames {
		for _, modelInfo := range modelMap.ModelInfoMap[modelDirName] {
			if !modelInfo.RunCi || len(modelInfo.BuildFiles) == 0 {
				// Skip CI in these cases
				continue
			} else if disabledModelPaths[modelDirName] {
				g.PostLabel("skipped: "+modelDirName, commonci.LabelColors["orange"], owner, repo, prNumber)
				continue
			}
			outputFile := filepath.Join(folderPath, fmt.Sprintf("%s==%s==pass", modelDirName, modelInfo.Name))
			failFile := filepath.Join(folderPath, fmt.Sprintf("%s==%s==fail", modelDirName, modelInfo.Name))
			builder.WriteString(fmt.Sprintf(fmtStr, modelMap.ModelRoot, commonci.RootDir, strings.Join(modelInfo.BuildFiles, " "), outputFile, outputFile, failFile))
		}
	}

	return builder.String()
}

func postPendingStatuses(g *commonci.GithubRequestHandler, validatorId string, versions []string, prApproved bool) []error {
	var errs []error
	validator := commonci.Validators[validatorId]
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
	flag.Parse()
	repoSplit := strings.Split(repoSlug, "/")
	owner = repoSplit[0]
	repo = repoSplit[1]
	if commitSHA == "" {
		log.Fatalf("no commit SHA")
	}
	if prNumber == 0 {
		log.Fatalf("no PR number")
	}

	modelMap, err := parseModels(modelRoot)
	if err != nil {
		log.Fatalf("CI flow failed due to error encountered while parsing spec files, parseModels: %v", err)
	}

	h := commonci.NewGitHubRequestHandler()

	prApproved, err := h.IsPRApproved(owner, repo, prNumber)
	if err != nil {
		log.Fatalf("warning: Could not check PR approved status, running all checks: %v", err)
		prApproved = true
	}

	versionsToRun := map[string][]string{}
	for validatorId, validator := range commonci.Validators {
		switch validatorId {
		case "pyang":
			versionsToRun["pyang"] = append([]string{""}, strings.Split(extraPV, ",")...)
		default:
			// Empty string is the "head" version, which is always run.
			versionsToRun[validatorId] = []string{""}
		}

		if errs := postPendingStatuses(h, validatorId, versionsToRun[validatorId], prApproved); errs != nil {
			log.Fatal(errs)
		}

		if !prApproved && !validator.RunBeforeApproval {
			// We don't run less important and long tests until PR is approved.
			continue
		}
		// Generate validation commands for the validator.
		for _, version := range versionsToRun[validatorId] {
			validatorResultPath := filepath.Join(commonci.ResultsDir, validatorId+version)
			if err := os.MkdirAll(validatorResultPath, 0644); err != nil {
				log.Fatalf("error while creating directory %q: %v", validatorResultPath, err)
			}
			log.Printf("Created results directory %q", validatorResultPath)

			scriptStr := genOpenConfigLinterCmd(h, validatorId, validatorResultPath, modelMap)
			scriptPath := filepath.Join(validatorResultPath, commonci.ScriptFileName)
			if err := ioutil.WriteFile(scriptPath, []byte(scriptStr), 0744); err != nil {
				log.Printf("error while writing script to path %q: %v", scriptPath, err)
			}
		}
	}

	for validatorId, versions := range versionsToRun {
		extraVersionFile := filepath.Join(commonci.ResultsDir, fmt.Sprintf("extra-%s-versions.txt", validatorId))
		if err := ioutil.WriteFile(extraVersionFile, []byte(strings.Join(versions, " ")), 0444); err != nil {
			log.Fatalf("error while writing extra versions file %q", extraVersionFile)
		}
	}

	// if err := h.DeleteLabel("test:labelTest", owner, repo, prNumber); err != nil {
	// 	log.Fatalf("error while deleting label: %v", err)
	// }
	// if err := h.PostLabel("test:labelTest2", "ffe200", owner, repo, prNumber); err != nil {
	// 	log.Fatalf("error while posting label: %v", err)
	// }
}
