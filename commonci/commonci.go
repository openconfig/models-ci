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

package commonci

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// commonci contains definitions and constants common to the CI process in
// general (esp. cmd_gen and post_result scripts).

const (
	// RootDir is the base directory of the CI, which in GCB is /workspace.
	RootDir = "/workspace"
	// ResultsDir contains all results of the CI process.
	ResultsDir = "/workspace/results"
	// UserConfigDir by convention contains the user config that is
	// passed from cmd_gen to later stages of the CI. It is common to all
	// CI steps.
	UserConfigDir = "/workspace/user-config"
	// CompatReportValidatorsFile notifies later CI steps of the validators
	// that should be reported as a compatibility report.
	CompatReportValidatorsFile = UserConfigDir + "/compat-report-validators.txt"
	// ForkSlugFile is created by cmd_gen to store the fork slug, if
	// present, for later CI steps.
	ForkSlugFile = UserConfigDir + "/fork-slug.txt"
	// ScriptFileName by convention is the script with the validator commands.
	ScriptFileName = "script.sh"
	// LatestVersionFileName by convention contains the version description
	// of the tool as output by the tool during the build.
	// Whenever the "latest" version of a tool has a version, it should
	// exist, and for now, it should be output into this file for display.
	LatestVersionFileName = "latest-version.txt"
	// OutFileName by convention contains the stdout of the script file.
	OutFileName = "out"
	// FailFileName by convention contains the stderr of the script file.
	FailFileName = "fail"
	// BadgeUploadCmdFile is output by post_results to upload the correct
	// status badge to GCS.
	BadgeUploadCmdFile = "upload-badge.sh"
)

// BoolStatusToString converts a pass/fail status from bool to string.
func BoolStatusToString(status bool) string {
	switch status {
	case true:
		return "pass"
	case false:
		return "fail"
	}
	return ""
}

// Emoji returns HTML for the emoji corresponding to a given status.
// If the status is not recognized, it returns an empty string.
func Emoji(status string) string {
	switch status {
	case "pass":
		return "&#x2705;" // checkmark emoji
	case "fail":
		return "&#x26D4;" // blocked emoji
	case "cmd":
		return "&#x1F4B2;" // dollar-sign emoji
	}
	return ""
}

// AppendVersionToName appends the version to the given validator name
func AppendVersionToName(validatorName, version string) string {
	if version != "" {
		version = "@" + version
	}
	return validatorName + version
}

// ValidatorResultsDir determines where a particular validator and version's
// results are
// stored.
func ValidatorResultsDir(validatorId, version string) string {
	return filepath.Join(ResultsDir, AppendVersionToName(validatorId, version))
}

// Validator describes a validation tool.
type Validator struct {
	// The longer name of the validator.
	Name string
	// IsPerModel means the validator is run per-model, not across the
	// entire repo of YANG files.
	IsPerModel bool
	// IgnoreRunCi says that the validator's commands should be generated
	// regardless of what the "run-ci" value in the .spec.yml is -- namely,
	// that it is a per-build validator, and bypasses the "run-ci" flag
	// that turns on more advanced testing.
	IgnoreRunCi bool
	// ReportOnly indicates that it's not itself a validator, it's just a
	// CI item that does reporting on other validators.
	ReportOnly bool
	// IsWidelyUsedTool indicates that the tool is a widely used tool whose
	// status should be reported on the front page of the repository.
	IsWidelyUsedTool bool
	// SupportedVersion is the lowest version supported to run in CI for
	// the validator. If empty, then all versions are supported.
	SupportedVersion string
}

// StatusName determines the status description for the version of the validator.
func (v *Validator) StatusName(version string) string {
	if v == nil {
		return ""
	}
	return AppendVersionToName(v.Name, version)
}

var (
	// Validators contains the set of supported validators to be run under CI.
	// The key is a unique identifier that's safe to use as a directory name.
	Validators = map[string]*Validator{
		"pyang": {
			Name:             "pyang",
			IsPerModel:       true,
			IsWidelyUsedTool: true,
			SupportedVersion: "2.2",
		},
		"oc-pyang": {
			Name:             "OpenConfig Linter",
			IsPerModel:       true,
			IsWidelyUsedTool: true,
		},
		"pyangbind": {
			Name:             "pyangbind",
			IsPerModel:       true,
			IsWidelyUsedTool: true,
		},
		"goyang-ygot": {
			Name:             "goyang/ygot",
			IsPerModel:       true,
			IsWidelyUsedTool: true,
		},
		"ygnmi": {
			Name:             "ygnmi",
			IsPerModel:       true,
			IsWidelyUsedTool: true,
		},
		"yanglint": {
			Name:             "yanglint",
			IsPerModel:       true,
			IsWidelyUsedTool: true,
		},
		"confd": {
			Name:             "ConfD Basic",
			IsPerModel:       true,
			IsWidelyUsedTool: true,
		},
		"regexp": {
			Name:       "regexp tests",
			IsPerModel: false,
		},
		"misc-checks": {
			Name:        "Miscellaneous Checks",
			IsPerModel:  true,
			IgnoreRunCi: true,
		},
		// This is a report-only entry for all validators configured to
		// report as a compatibility check instead of as a standalone
		// PR status.
		"compat-report": {
			Name:       "Compatibility Report",
			IsPerModel: false,
			ReportOnly: true,
		},
	}

	// LabelColors are some helper hex colours for posting to GitHub.
	LabelColors = map[string]string{
		"yellow": "ffe200",
		"red":    "ff0000",
		"orange": "ffa500",
		"blue":   "00bfff",
	}

	// validStatuses are the valid pull request status codes that are valid in the GitHub UI.
	validStatuses = map[string]bool{
		"pending": true,
		"success": true,
		"error":   true,
		"failure": true,
	}
)

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

// SingleLineBuildFiles returns all of the build files defined by all the
// .spec.yml files in the models, if run-ci is true, as a single,
// space-separated line.
func (m OpenConfigModelMap) SingleLineBuildFiles() string {
	modelDirNames := make([]string, 0, len(m.ModelInfoMap))
	for modelDirName := range m.ModelInfoMap {
		modelDirNames = append(modelDirNames, modelDirName)
	}
	sort.Strings(modelDirNames)

	var buildFiles []string
	for _, modelDirName := range modelDirNames {
		fmt.Println(modelDirName)
		for _, modelInfo := range m.ModelInfoMap[modelDirName] {
			if !modelInfo.RunCi {
				continue
			}
			buildFiles = append(buildFiles, modelInfo.BuildFiles...)
		}
	}
	return strings.Join(buildFiles, " ")
}

// ParseOCModels walks the path given at modelRoot to populate the OpenConfigModelMap.
func ParseOCModels(modelRoot string) (OpenConfigModelMap, error) {
	modelInfoMap := map[string][]ModelInfo{}
	err := filepath.Walk(modelRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v", path, err)
		}
		if !info.IsDir() && info.Name() == ".spec.yml" {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open spec file at path %q: %v", path, err)
			}
			m := []ModelInfo{}
			if err := yaml.NewDecoder(file).Decode(&m); err != nil {
				return fmt.Errorf("error while unmarshalling spec file at path %q: %v", path, err)
			}

			// Change the build paths to the absolute correct paths.
			for _, info := range m {
				for i, fileName := range info.BuildFiles {
					info.BuildFiles[i] = filepath.Join(modelRoot, strings.TrimPrefix(fileName, "yang/"))
				}
			}

			relPath, err := filepath.Rel(modelRoot, filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("failed to calculate relpath at path %q (modelRoot %q): %v", path, modelRoot, err)
			}
			// Allow nested model directories to be used later on as a partial file name.
			relPath = strings.ReplaceAll(relPath, "/", ":")
			modelInfoMap[relPath] = m
		}
		return nil
	})

	return OpenConfigModelMap{ModelRoot: modelRoot, ModelInfoMap: modelInfoMap}, err
}

type ValidatorAndVersion struct {
	ValidatorId string
	Version     string
}

// GetValidatorAndVersionsFromString converts a comma-separated list of
// <validatorId>@<version> names to a list of ValidatorAndVersion and nested
// map of validatorId to version for checking existence.
func GetValidatorAndVersionsFromString(validatorsAndVersionsStr string) ([]ValidatorAndVersion, map[string]map[string]bool) {
	var compatValidators []ValidatorAndVersion
	compatValidatorsMap := map[string]map[string]bool{}
	for _, vvStr := range strings.Fields(strings.ReplaceAll(validatorsAndVersionsStr, ",", " ")) {
		vvSegments := strings.SplitN(vvStr, "@", 2)
		vv := ValidatorAndVersion{ValidatorId: vvSegments[0]}
		if len(vvSegments) == 2 {
			vv.Version = vvSegments[1]
		}
		m, ok := compatValidatorsMap[vv.ValidatorId]
		if !ok {
			m = map[string]bool{}
			compatValidatorsMap[vv.ValidatorId] = m
		}
		// De-dup validator@version names.
		if !m[vv.Version] {
			compatValidators = append(compatValidators, vv)
			m[vv.Version] = true
		}
	}
	return compatValidators, compatValidatorsMap
}

// ValidatorAndVersionsDiff removes the comma-separated list of
// <validatorId>@<version> entries in bStr from aStr.
func ValidatorAndVersionsDiff(aStr, bStr string) string {
	aVVs, _ := GetValidatorAndVersionsFromString(aStr)
	_, bVVMap := GetValidatorAndVersionsFromString(bStr)
	var remainingVVs []string
	for _, vv := range aVVs {
		if !bVVMap[vv.ValidatorId][vv.Version] {
			remainingVVs = append(remainingVVs, AppendVersionToName(vv.ValidatorId, vv.Version))
		}
	}
	return strings.Join(remainingVVs, ",")
}
