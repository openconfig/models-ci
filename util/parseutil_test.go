// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"google.golang.org/protobuf/testing/protocmp"

	pb "github.com/openconfig/models-ci/proto/results"
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

func TestParsePyangTextprotoOutput(t *testing.T) {
	tests := []struct {
		desc          string
		in            string
		want          *pb.PyangOutput
		wantErrSubstr string
	}{{
		desc: "single error line",
		in:   `messages:{path:"a.yang" line:30 code:"DUPLICATE_CHILD_NAME" type:"error" level:1 message:'there is already a child node to "cc" at a.yang:27 with the name "ccc" defined at a.yang:28'}`,
		want: &pb.PyangOutput{
			Messages: []*pb.PyangMessage{{
				Path:    "a.yang",
				Line:    30,
				Code:    "DUPLICATE_CHILD_NAME",
				Type:    "error",
				Level:   1,
				Message: `there is already a child node to "cc" at a.yang:27 with the name "ccc" defined at a.yang:28`,
			}},
		},
	}, {
		desc: "empty line",
		in:   ``,
		want: &pb.PyangOutput{},
	}, {
		desc:          "unrecognized line",
		in:            `foo`,
		wantErrSubstr: "unrecognized line",
	}, {
		desc: "error line and warning lines",
		in: `messages:{path:"tmp/a.yang" line:15 code:"UNEXPECTED_KEYWORD" type:"error" level:1 message:'unexpected keyword "description"'}
messages:{path:"tmp/a.yang" line:26 code:"LONG_LINE" type:"warning" level:4 message:'line length 17 exceeds 5 characters'}
messages:{path:"tmp/a.yang" line:30 code:"DUPLICATE_CHILD_NAME" type:"error" level:1 message:'there is already a child node to "cc" at tmp/a.yang:27 with the name "ccc" defined at tmp/a.yang:28'}
messages:{path:"/workspace/yang/isis/openconfig-isis.yang" line:27 code:"LINT_BAD_REVISION" type:"error" level:3 message:'RFC 6087: 4.6: the module's revision 2021-03-17 is older than submodule openconfig-isis-lsp's revision 2021-06-16'}`,
		want: &pb.PyangOutput{
			Messages: []*pb.PyangMessage{{
				Path:    "tmp/a.yang",
				Line:    15,
				Code:    "UNEXPECTED_KEYWORD",
				Type:    "error",
				Level:   1,
				Message: `unexpected keyword "description"`,
			}, {
				Path:    "tmp/a.yang",
				Line:    26,
				Code:    "LONG_LINE",
				Type:    "warning",
				Level:   4,
				Message: `line length 17 exceeds 5 characters`,
			}, {
				Path:    "tmp/a.yang",
				Line:    30,
				Code:    "DUPLICATE_CHILD_NAME",
				Type:    "error",
				Level:   1,
				Message: `there is already a child node to "cc" at tmp/a.yang:27 with the name "ccc" defined at tmp/a.yang:28`,
			}, {
				Path:  "/workspace/yang/isis/openconfig-isis.yang",
				Line:  27,
				Code:  "LINT_BAD_REVISION",
				Type:  "error",
				Level: 3,
				// This tests escaping single quotes that are in the error message.
				Message: `RFC 6087: 4.6: the module's revision 2021-03-17 is older than submodule openconfig-isis-lsp's revision 2021-06-16`,
			}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ParsePyangTextprotoOutput(tt.in)
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(got, tt.want, protocmp.Transform()); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
		})
	}
}
