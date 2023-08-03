// Copyright 2023 Google Inc.
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

package ocdiff

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/openconfig/ygot/testutil"
)

var updateGolden = flag.Bool("update_golden", false, "Update golden files")

func getAllYANGFiles(t *testing.T, path string) []string {
	t.Helper()
	var files []string
	if err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(info.Name()) == ".yang" {
				files = append(files, path)
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestDiffReport(t *testing.T) {
	tests := []struct {
		name     string
		inOpts   []Option
		wantFile string
	}{{
		name:     "no-options",
		wantFile: "testdata/no-options.txt",
	}, {
		name: "github-comment",
		inOpts: []Option{
			WithGithubCommentStyle(),
		},
		wantFile: "testdata/github-comment.txt",
	}, {
		name: "disallowed-incompats",
		inOpts: []Option{
			WithDisallowedIncompatsOnly(),
		},
		wantFile: "testdata/disallowed-incompats.txt",
	}, {
		name: "github-comment-disallowed-incompats",
		inOpts: []Option{
			WithGithubCommentStyle(),
			WithDisallowedIncompatsOnly(),
		},
		wantFile: "testdata/github-comment-disallowed-incompats.txt",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := NewDiffReport([]string{"testdata/yang/incl"}, []string{"testdata/yang/incl"}, getAllYANGFiles(t, "testdata/yang/old"), getAllYANGFiles(t, "testdata/yang/new"))
			if err != nil {
				t.Fatal(err)
			}
			gotReport := report.Report(tt.inOpts...)
			wantFileBytes, rferr := os.ReadFile(tt.wantFile)
			if rferr != nil {
				t.Fatalf("os.ReadFile(%q) error: %v", tt.wantFile, rferr)
			}

			if wantReport := string(wantFileBytes); gotReport != wantReport {
				if *updateGolden {
					if err := os.WriteFile(tt.wantFile, []byte(gotReport), 0644); err != nil {
						t.Fatal(err)
					}
				}
				// Use difflib to generate a unified diff between the
				// two code snippets such that this is simpler to debug
				// in the test output.
				diff, _ := testutil.GenerateUnifiedDiff(wantReport, gotReport)
				t.Errorf("did not return correct report (file: %v), diff:\n%s", tt.wantFile, diff)
			}
		})
	}
}
