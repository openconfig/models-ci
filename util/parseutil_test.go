package util

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseStandardOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want StandardOutput
	}{{
		name: "only warnings with subpath",
		in: `/workspace/release/yang/acl/openconfig-packet-match-types.yang:1: warning: Module openconfig-packet-match-types is missing a grouping suffixed with -top
/workspace/release/yang/openconfig-extensions.yang:49 (at /workspace/release/yang/bfd/openconfig-bfd.yang:226): warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:forty-nine (at /workspace/release/yang/bfd/openconfig-bfd.yang:226): warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:158: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
foo
/workspace/release/yang/openconfig-extensions.yang:169: error: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/types/openconfig-inet-types.yang:1: warning: Module openconfig-inet-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-types.yang:1: error: Module openconfig-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-yang-types.yang:1: warning: Module openconfig-yang-types is missing a grouping suffixed with -top
`,
		want: StandardOutput{
			ErrorLines: []*StandardErrorLine{{
				Path:    "/workspace/release/yang/openconfig-extensions.yang",
				LineNo:  169,
				Status:  "error",
				Message: `RFC 6087: 4.3: statement "yin-element" is given with its default value "false"`,
			}, {
				Path:    "/workspace/release/yang/types/openconfig-types.yang",
				LineNo:  1,
				Status:  "error",
				Message: `Module openconfig-types is missing a grouping suffixed with -top`,
			}},
			WarningLines: []*StandardErrorLine{{
				Path:    "/workspace/release/yang/acl/openconfig-packet-match-types.yang",
				LineNo:  1,
				Status:  "warning",
				Message: "Module openconfig-packet-match-types is missing a grouping suffixed with -top",
			}, {
				Path:    "/workspace/release/yang/openconfig-extensions.yang",
				LineNo:  49,
				Status:  "warning",
				Message: `RFC 6087: 4.3: statement "yin-element" is given with its default value "false"`,
			}, {
				Path:    "/workspace/release/yang/openconfig-extensions.yang",
				LineNo:  158,
				Status:  "warning",
				Message: `RFC 6087: 4.3: statement "yin-element" is given with its default value "false"`,
			}, {
				Path:    "/workspace/release/yang/types/openconfig-inet-types.yang",
				LineNo:  1,
				Status:  "warning",
				Message: `Module openconfig-inet-types is missing a grouping suffixed with -top`,
			}, {
				Path:    "/workspace/release/yang/types/openconfig-yang-types.yang",
				LineNo:  1,
				Status:  "warning",
				Message: `Module openconfig-yang-types is missing a grouping suffixed with -top`,
			}},
			OtherLines: []string{
				`/workspace/release/yang/openconfig-extensions.yang:forty-nine (at /workspace/release/yang/bfd/openconfig-bfd.yang:226): warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"`,
				`foo`,
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, ParseStandardOutput(tt.in)); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}
