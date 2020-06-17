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
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/openconfig/models-ci/commonci"
)

var (
	// Commandline flags: should be string if it may not exist
	modelRoot          string // modelRoot is the root directory of the models.
	repoSlug           string // repoSlug is the "owner/repo" name of the models repo (e.g. openconfig/public).
	prHeadRepoURL      string // prHeadRepoURL is the URL of the HEAD repo for PRs (e.g. https://github.com/openconfig/public).
	commitSHA          string
	branchName         string // branchName is the name of the branch where the commit occurred.
	prNumberStr        string // prNumberStr is the PR number.
	compatReports      string // e.g. "goyang-ygot,pyangbind,pyang@1.7.8"
	extraPyangVersions string // e.g. "1.2.3,3.4.5"
	skippedValidators  string // e.g. "yanglint,pyang@head"

	// Derived flags (for ease of use)
	owner     string
	repo      string
	prNumber  int
	headOwner string
	headRepo  string

	// local run flags
	local             bool   // local run toggle
	localResultsDir   string // folder into which the command outputs its results
	localValidatorId  string
	localModelDirName string // a model directory (e.g. network-instance, aft)

	// Miscellaneous flags
	listBuildFiles bool // Show all build files from the .spec.yml files as a single line.

	// disabledModelPaths are the paths whose models should not undergo CI.
	// These should be temporary -- they're only here to help the transition to CI.
	// To represent a multi-level directory, use ":" instead of "/" as the delimiter.
	disabledModelPaths = map[string]bool{
		"wifi:access-points": false,
		"wifi:ap-manager":    false,
		"wifi:mac":           false,
		"wifi:phy":           false,
	}
)

func init() {
	// GCB-required flags
	flag.StringVar(&modelRoot, "modelRoot", "", "root directory to OpenConfig models")
	flag.StringVar(&repoSlug, "repo-slug", "", "repo where CI is run")
	flag.StringVar(&prHeadRepoURL, "pr-head-repo-url", "", "PR head repo URL")
	flag.StringVar(&commitSHA, "commit-sha", "", "commit SHA of the PR")
	flag.StringVar(&prNumberStr, "pr-number", "", "PR number")
	flag.StringVar(&branchName, "branch", "", "branch name of commit")
	flag.StringVar(&compatReports, "compat-report", "", "comma-separated validators (e.g. goyang-ygot,pyang@1.7.8,pyang@head) in compatibility report instead of a standalone PR status")
	flag.StringVar(&skippedValidators, "skipped-validators", "", "comma-separated validators (e.g. goyang-ygot,pyang@1.7.8,pyang@head) not to be ran at all, not even in the compatibility report")
	flag.StringVar(&extraPyangVersions, "extra-pyang-versions", "", "comma-separated extra pyang versions to run")

	// Local run flags
	flag.BoolVar(&local, "local", false, "use with validator, modelDirName, resultsDir to get a particular model's command")
	flag.StringVar(&localResultsDir, "resultsDir", "~/tmp/ci-results", "root directory to OpenConfig models")
	flag.StringVar(&localValidatorId, "validator", "", "")
	flag.StringVar(&localModelDirName, "modelDirName", "", "")

	// Miscellaneous flags
	flag.BoolVar(&listBuildFiles, "listBuildFiles", false, "Show all build files from the .spec.yml files as a single line.")
}

// mustTemplate generates a template.Template for a particular named source template
func mustTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Parse(src))
}

type cmdParams struct {
	ModelRoot    string
	RepoRoot     string
	BuildFiles   []string
	ModelDirName string
	ModelName    string
	ResultsDir   string
}

var (
	pyangCmdTemplate = mustTemplate("pyang", `if ! $@ -p {{ .ModelRoot }} -p {{ .RepoRoot }}/third_party/ietf {{- range $i, $buildFile := .BuildFiles }} {{ $buildFile }} {{- end }} &> {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass; then
  mv {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==fail
fi &
`)

	ocPyangCmdTemplate = mustTemplate("oc-pyang", `if ! $@ -p {{ .ModelRoot }} -p {{ .RepoRoot }}/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH {{- range $i, $buildFile := .BuildFiles }} {{ $buildFile }} {{- end }} &> {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass; then
  mv {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==fail
fi &
`)

	pyangbindCmdTemplate = mustTemplate("pyangbind", `if ! $@ -p {{ .ModelRoot }} -p {{ .RepoRoot }}/third_party/ietf -f pybind -o {{ .ModelDirName }}.{{ .ModelName }}.binding.py {{- range $i, $buildFile := .BuildFiles }} {{ $buildFile }} {{- end }} &> {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass; then
  mv {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==fail
fi &
`)

	goyangYgotCmdTemplate = mustTemplate("goyang-ygot", `if ! /go/bin/generator \
-path={{ .ModelRoot }},{{ .RepoRoot }}/third_party/ietf \
-output_file=`+commonci.ResultsDir+`/goyang-ygot/{{ .ModelDirName }}.{{ .ModelName }}.oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
{{ range $i, $buildFile := .BuildFiles -}} {{ $buildFile }} {{ end -}} &> {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass; then
  mv {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==fail
fi &
`)

	yanglintCmdTemplate = mustTemplate("yanglint", `if ! yanglint -p {{ .ModelRoot }} -p {{ .RepoRoot }}/third_party/ietf {{- range $i, $buildFile := .BuildFiles }} {{ $buildFile }} {{- end }} &> {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass; then
  mv {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==fail
fi
`)

	confdCmdTemplate = mustTemplate("confd", `status=0
{{- range $i, $buildFile := .BuildFiles }}
$1 -c --yangpath $2 {{ $buildFile }} &>> {{ $.ResultsDir }}/{{ $.ModelDirName }}=={{ $.ModelName }}==pass || status=1
{{- end }}
if [[ $status -eq "1" ]]; then
  mv {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==pass {{ .ResultsDir }}/{{ .ModelDirName }}=={{ .ModelName }}==fail
fi
`)

	miscChecksCmdTemplate = mustTemplate("misc-checks", `if ! /go/bin/ocversion -p {{ .ModelRoot }},{{ .RepoRoot }}/third_party/ietf {{- range $i, $buildFile := .BuildFiles }} {{ $buildFile }} {{- end }} > {{ .ResultsDir }}/{{ .ModelDirName }}.{{ .ModelName }}.pr-file-parse-log; then
  >&2 echo "parse of {{ .ModelDirName }}.{{ .ModelName }} reported non-zero status."
fi
`)
)

// validatorTemplate creates the customized format string to invoke the
// given validator on a model.
func validatorTemplate(validatorId string) (*template.Template, error) {
	switch validatorId {
	case "pyang":
		return pyangCmdTemplate, nil
	case "oc-pyang":
		return ocPyangCmdTemplate, nil
	case "pyangbind":
		return pyangbindCmdTemplate, nil
	case "goyang-ygot":
		return goyangYgotCmdTemplate, nil
	case "yanglint":
		return yanglintCmdTemplate, nil
	case "confd":
		return confdCmdTemplate, nil
	case "misc-checks":
		return miscChecksCmdTemplate, nil
	}
	return nil, fmt.Errorf("validatorTemplate: unrecognized validatorId for creating a per-model command %q", validatorId)
}

// genValidatorCommandForModelDir generates the validator command for a single modelDir.
func genValidatorCommandForModelDir(validatorId, resultsDir, modelDirName string, modelMap commonci.OpenConfigModelMap) (string, error) {
	var builder strings.Builder
	cmdTemplate, err := validatorTemplate(validatorId)
	if err != nil {
		return "", err
	}
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return "", err
	}
	for _, modelInfo := range modelMap.ModelInfoMap[modelDirName] {
		// First check whether to skip CI.
		if len(modelInfo.BuildFiles) == 0 || (!modelInfo.RunCi && !validator.IgnoreRunCi) {
			continue
		}
		if err := cmdTemplate.Execute(&builder, &cmdParams{
			ModelRoot:    modelMap.ModelRoot,
			RepoRoot:     commonci.RootDir,
			BuildFiles:   modelInfo.BuildFiles,
			ModelDirName: modelDirName,
			ModelName:    modelInfo.Name,
			ResultsDir:   resultsDir,
		}); err != nil {
			return "", err
		}
	}
	return builder.String(), nil
}

// labelPoster is an interface with just a function for posting a GitHub label to a PR.
type labelPoster interface {
	PostLabel(labelName, labelColor, owner, repo string, prNumber int) error
}

// genOpenConfigValidatorScript generates the whole validation script for the given validator.
// Tool version should be "" unless a non-latest version is used.
// Scripts generated by this function assume the following:
//  1. Each validator uses a different command which can be customized, but all
//     will be run only on a single model as specified in the .spec.yml file.
//  2. Thus, a validation command and result is provided for each model.
//  3. A file indicating pass/fail is output for each model into the given result directory.
// Files names follow the "modelDir==model==status" format with no file extensions.
// The local flag indicates to run this as a helper to generate the script,
// rather than running it within GCB.
func genOpenConfigValidatorScript(g labelPoster, validatorId, version string, modelMap commonci.OpenConfigModelMap) (string, error) {
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
			if prNumber != 0 {
				g.PostLabel("skipped: "+modelDirName, commonci.LabelColors["orange"], owner, repo, prNumber)
			}
			continue
		}
		cmdStr, err := genValidatorCommandForModelDir(validatorId, resultsDir, modelDirName, modelMap)
		if err != nil {
			return "", err
		}
		builder.WriteString(cmdStr)
	}

	// In case there are parallel commands.
	builder.WriteString("wait\n")
	return builder.String(), nil
}

// postInitialStatus posts the initial status for all versions of a validator.
func postInitialStatus(g *commonci.GithubRequestHandler, validatorId string, version string) error {
	validator, ok := commonci.Validators[validatorId]
	if !ok {
		return fmt.Errorf("validator %q not recognized", validatorId)
	}
	validatorName := validator.StatusName(version)
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

	if err := g.UpdatePRStatus(update); err != nil {
		log.Printf("error: couldn't update PR: %s", err)
		log.Printf("GithubPRUpdate: %+v", update)
		return err
	}
	return nil
}

func main() {
	// Parse derived flags.
	flag.Parse()

	if modelRoot == "" {
		log.Fatalf("Must supply modelRoot path")
	}
	// Populate information necessary for validation script generation.
	modelMap, err := commonci.ParseOCModels(modelRoot)
	if err != nil {
		log.Fatalf("CI flow failed due to error encountered while parsing spec files, commonci.ParseOCModels: %v", err)
	}

	if listBuildFiles {
		fmt.Println(modelMap.SingleLineBuildFiles())
		return
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

	if prNumberStr != "" {
		var err error
		if prNumber, err = strconv.Atoi(prNumberStr); err != nil {
			log.Fatalf("error encountered while parsing PR number: %s", err)
		}
	}

	badgeOnly := false
	// If it's a push on master, just upload badge for normal validators as the only action.
	if prNumber == 0 {
		if branchName != "master" {
			log.Fatalf("cmd_gen: There is no action to take for a non-master branch push, please re-examine your push triggers")
		}
		badgeOnly = true
	}

	// Skip testing non-widely used validators, as we don't need to post badges for those tools.
	if badgeOnly {
		for validatorId, validator := range commonci.Validators {
			if !validator.IsWidelyUsedTool {
				// Here we assume simply that non widely-used checks don't have a version specified.
				skippedValidators += "," + validatorId
			}
		}
	}

	if err := os.MkdirAll(commonci.ResultsDir, 0644); err != nil {
		log.Fatalf("error while creating directory %q: %v", commonci.ResultsDir, err)
	}
	if err := os.MkdirAll(commonci.UserConfigDir, 0644); err != nil {
		log.Fatalf("error while creating directory %q: %v", commonci.UserConfigDir, err)
	}

	repoSplit := strings.Split(repoSlug, "/")
	owner = repoSplit[0]
	repo = repoSplit[1]
	if commitSHA == "" {
		log.Fatalf("no commit SHA")
	}

	headOwner = owner
	headRepo = repo
	if prHeadRepoURL != "" {
		// Expected format: e.g. https://github.com/openconfig/public
		URLSplit := strings.Split(prHeadRepoURL, "/")
		headOwner = URLSplit[len(URLSplit)-2]
		headRepo = URLSplit[len(URLSplit)-1]
		if headOwner != owner || headRepo != repo {
			remoteBranch := headOwner + "/" + headRepo
			// If this is a fork, let later CI steps know the fork repo slug.
			if err := ioutil.WriteFile(commonci.ForkSlugFile, []byte(remoteBranch), 0444); err != nil {
				log.Fatalf("error while writing fork slug file %q: %v", commonci.ForkSlugFile, err)
			}
			log.Printf("fork detected for remote repo %q", remoteBranch)
		}
	}

	compatReports = commonci.ValidatorAndVersionsDiff(compatReports, skippedValidators)
	if !badgeOnly {
		// Notify later CI steps of the validators that should be reported as a compatibility report.
		if err := ioutil.WriteFile(commonci.CompatReportValidatorsFile, []byte(compatReports), 0444); err != nil {
			log.Fatalf("error while writing compatibility report validators file %q: %v", commonci.CompatReportValidatorsFile, err)
		}
	}

	_, compatValidatorsMap := commonci.GetValidatorAndVersionsFromString(compatReports)
	_, skippedValidatorsMap := commonci.GetValidatorAndVersionsFromString(skippedValidators)

	// Generate validation scripts, files, and post initial status on GitHub.
	h, err := commonci.NewGitHubRequestHandler()
	if err != nil {
		log.Fatal(err)
	}
	for validatorId, validator := range commonci.Validators {
		if validator.ReportOnly {
			continue
		}

		var extraVersions []string
		if validatorId == "pyang" {
			// pyang also runs a HEAD version.
			extraVersions = strings.Split(extraPyangVersions, ",")
		}
		// Write a list of the extra validator versions into the
		// designated extra versions file in order to be relayed to the
		// corresponding test.sh (next stage of the CI pipeline).
		if len(extraVersions) > 0 {
			extraVersionFile := filepath.Join(commonci.UserConfigDir, fmt.Sprintf("extra-%s-versions.txt", validatorId))
			if err := ioutil.WriteFile(extraVersionFile, []byte(strings.Join(extraVersions, " ")), 0444); err != nil {
				log.Fatalf("error while writing extra versions file %q: %v", extraVersionFile, err)
			}
		}

		// Empty string means the latest version, which is always run.
		versionsToRun := append([]string{""}, extraVersions...)
		if validatorId == "pyang" {
			versionsToRun = append(versionsToRun, "head")
		}

		// Generate validation commands for the validator.
		for _, version := range versionsToRun {
			if skippedValidatorsMap[validatorId][version] {
				log.Printf("Not activating skipped validator %s", commonci.AppendVersionToName(validatorId, version))
				continue
			}

			// Post initial PR status.
			if !badgeOnly && !compatValidatorsMap[validatorId][version] {
				if errs := postInitialStatus(h, validatorId, version); errs != nil {
					log.Fatal(errs)
				}
			}

			// Create results dir, which activates the validator script.
			validatorResultsDir := commonci.ValidatorResultsDir(validatorId, version)
			if err := os.MkdirAll(validatorResultsDir, 0644); err != nil {
				log.Fatalf("error while creating directory %q: %v", validatorResultsDir, err)
			}
			log.Printf("Created results directory %q", validatorResultsDir)

			if !validator.IsPerModel {
				// We don't generate commands when the tool is
				// ran directly on the entire models directory.
				// (i.e. a repo-level validator)
				continue
			}

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
