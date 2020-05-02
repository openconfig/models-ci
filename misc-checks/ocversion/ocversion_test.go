package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPrintOCVersions(t *testing.T) {
	tests := []struct {
		desc    string
		inPath  []string
		inFiles []string
		want    string
		wantErr bool
	}{{
		desc:    "single extension",
		inPath:  []string{"testdata"},
		inFiles: []string{"testdata/openconfig-single-extension.yang"},
		want: `openconfig-extensions.yang:
openconfig-single-extension.yang: openconfig-version:"0.4.2"
`,
	}, {
		desc:    "multiple extensions",
		inPath:  []string{"testdata"},
		inFiles: []string{"testdata/openconfig-telemetry-types.yang"},
		want: `openconfig-extensions.yang:
openconfig-telemetry-types.yang: openconfig-version:"0.4.2"
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
		want: `openconfig-extensions.yang:
openconfig-use-other-extension.yang:
other-extensions.yang:
`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			entries, errs := buildModuleEntries(tt.inPath, tt.inFiles)
			if gotErr := errs != nil; gotErr != tt.wantErr {
				t.Fatal(errs)
			}

			var b bytes.Buffer
			printOCVersions(&b, entries)
			got, want := strings.Split(b.String(), "\n"), strings.Split(tt.want, "\n")
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
		})
	}
}
