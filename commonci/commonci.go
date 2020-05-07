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
)

// ValidatorResultsDir determines where a particular validator and version's
// results are
// stored.
func ValidatorResultsDir(validatorId, version string) string {
	return filepath.Join(ResultsDir, validatorId+"@"+version)
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
}

// StatusName determines the status description for the version of the validator.
func (v *Validator) StatusName(version string) string {
	if v == nil {
		return ""
	}
	return ValidatorResultsDir(v.Name, version)
}

var (
	// Validators contains the set of supported validators to be run under CI.
	// The key is a unique identifier that's safe to use as a directory name.
	Validators = map[string]*Validator{
		"pyang": &Validator{
			Name:       "Pyang",
			IsPerModel: true,
		},
		"oc-pyang": &Validator{
			Name:       "OpenConfig Linter",
			IsPerModel: true,
		},
		"pyangbind": &Validator{
			Name:       "Pyangbind",
			IsPerModel: true,
		},
		"goyang-ygot": &Validator{
			Name:       "goyang/ygot",
			IsPerModel: true,
		},
		"yanglint": &Validator{
			Name:       "yanglint",
			IsPerModel: true,
		},
		"regexp": &Validator{
			Name:       "regexp tests",
			IsPerModel: false,
		},
		"misc-checks": &Validator{
			Name:        "Miscellaneous Checks",
			IsPerModel:  true,
			IgnoreRunCi: true,
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
