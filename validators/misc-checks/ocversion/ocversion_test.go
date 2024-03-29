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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOcVersionsList(t *testing.T) {
	tests := []struct {
		desc    string
		inPath  []string
		inFiles []string
		want    string
		wantErr bool
	}{{
		desc:   "module and submodule",
		inPath: []string{"testdata"},
		inFiles: []string{
			"testdata/openconfig-extensions.yang",
			"testdata/openconfig-extensions-submodule.yang",
			"testdata/openconfig-single-extension.yang",
			"testdata/openconfig-single-extension-submodule.yang",
		},
		want: `openconfig-extensions.yang: belonging-module:"openconfig-extensions"
openconfig-extensions-submodule.yang: belonging-module:"openconfig-extensions" openconfig-version:"0.5.0"
openconfig-single-extension.yang: belonging-module:"openconfig-single-extension" openconfig-version:"0.4.2"
openconfig-single-extension-submodule.yang: belonging-module:"openconfig-single-extension" openconfig-version:"0.4.3"
`,
	}, {
		desc:    "multiple extensions",
		inPath:  []string{"testdata"},
		inFiles: []string{"testdata/openconfig-telemetry-types.yang"},
		want: `openconfig-extensions.yang: belonging-module:"openconfig-extensions"
openconfig-telemetry-types.yang: belonging-module:"openconfig-telemetry-types" openconfig-version:"0.4.2"
`,
	}, {
		desc:    "invalid file",
		inPath:  []string{"testdata"},
		inFiles: []string{"testdata/openconfig-invalid.yang"},
		wantErr: true,
	}, {
		desc:    "other-extensions module used for openconfig-extension value",
		inPath:  []string{"testdata"},
		inFiles: []string{"testdata/openconfig-use-other-extension.yang"},
		want: `openconfig-extensions.yang: belonging-module:"openconfig-extensions"
openconfig-use-other-extension.yang: belonging-module:"openconfig-use-other-extension"
other-extensions.yang: belonging-module:"other-extensions"
`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			entries, errs := buildModuleEntries(tt.inPath, tt.inFiles)
			if gotErr := errs != nil; gotErr != tt.wantErr {
				t.Fatal(errs)
			}

			got, want := strings.Split(ocVersionsList(entries), "\n"), strings.Split(tt.want, "\n")
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
		})
	}
}
