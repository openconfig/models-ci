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
	"github.com/openconfig/gnmi/errdiff"
)

func TestProcessAnyPyangOutput(t *testing.T) {
	modelRoot = "/workspace/release/yang"

	tests := []struct {
		name         string
		in           string
		inPass       bool
		inNoWarnings bool
		want         string
	}{{
		name: "only warnings with subpath",
		in: `/workspace/release/yang/acl/openconfig-packet-match-types.yang:1: warning: Module openconfig-packet-match-types is missing a grouping suffixed with -top
/workspace/release/yang/openconfig-extensions.yang:49 (at /workspace/release/yang/bfd/openconfig-bfd.yang:226): warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:158: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:169: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/types/openconfig-inet-types.yang:1: warning: Module openconfig-inet-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-types.yang:1: warning: Module openconfig-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-yang-types.yang:1: warning: Module openconfig-yang-types is missing a grouping suffixed with -top
`,
		inPass: true,
		want: `Passed.
<ul>
  <li>acl/openconfig-packet-match-types.yang (1): warning: <pre>Module openconfig-packet-match-types is missing a grouping suffixed with -top</pre></li>
  <li>openconfig-extensions.yang (49): warning: <pre>RFC 6087: 4.3: statement "yin-element" is given with its default value "false"</pre></li>
  <li>openconfig-extensions.yang (158): warning: <pre>RFC 6087: 4.3: statement "yin-element" is given with its default value "false"</pre></li>
  <li>openconfig-extensions.yang (169): warning: <pre>RFC 6087: 4.3: statement "yin-element" is given with its default value "false"</pre></li>
  <li>types/openconfig-inet-types.yang (1): warning: <pre>Module openconfig-inet-types is missing a grouping suffixed with -top</pre></li>
  <li>types/openconfig-types.yang (1): warning: <pre>Module openconfig-types is missing a grouping suffixed with -top</pre></li>
  <li>types/openconfig-yang-types.yang (1): warning: <pre>Module openconfig-yang-types is missing a grouping suffixed with -top</pre></li>
</ul>
`,
	}, {
		name: "warnings and errors, and prioritizing errors",
		in: `/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "A" should be of the form UPPERCASE_WITH_UNDERSCORES: A
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "G" should be of the form UPPERCASE_WITH_UNDERSCORES: G
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "N" should be of the form UPPERCASE_WITH_UNDERSCORES: N
/workspace/release/yang/openconfig-extensions.yang:49: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:158: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:169: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/types/openconfig-inet-types.yang:1: warning: Module openconfig-inet-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-types.yang:1: warning: Module openconfig-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-yang-types.yang:1: warning: Module openconfig-yang-types is missing a grouping suffixed with -top
/workspace/release/yang/vlan/openconfig-vlan-types.yang:1: warning: Module openconfig-vlan-types is missing a grouping suffixed with -top
/workspace/release/yang/wifi/types/openconfig-wifi-types.yang:1: warning: Module openconfig-wifi-types is missing a grouping suffixed with -top
/workspace/release/yang/wifi/types/openconfig-wifi-types.yang:288: error: identity name "BETTER-CHANNEL" should be of the form UPPERCASE_WITH_UNDERSCORES: "BETTER-CHANNEL"
`,
		inPass: false,
		want: `<ul>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "A" should be of the form UPPERCASE_WITH_UNDERSCORES: A</pre></li>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B</pre></li>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "G" should be of the form UPPERCASE_WITH_UNDERSCORES: G</pre></li>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "N" should be of the form UPPERCASE_WITH_UNDERSCORES: N</pre></li>
  <li>wifi/types/openconfig-wifi-types.yang (288): error: <pre>identity name "BETTER-CHANNEL" should be of the form UPPERCASE_WITH_UNDERSCORES: "BETTER-CHANNEL"</pre></li>
  <li>openconfig-extensions.yang (49): warning: <pre>RFC 6087: 4.3: statement "yin-element" is given with its default value "false"</pre></li>
  <li>openconfig-extensions.yang (158): warning: <pre>RFC 6087: 4.3: statement "yin-element" is given with its default value "false"</pre></li>
  <li>openconfig-extensions.yang (169): warning: <pre>RFC 6087: 4.3: statement "yin-element" is given with its default value "false"</pre></li>
  <li>types/openconfig-inet-types.yang (1): warning: <pre>Module openconfig-inet-types is missing a grouping suffixed with -top</pre></li>
  <li>types/openconfig-types.yang (1): warning: <pre>Module openconfig-types is missing a grouping suffixed with -top</pre></li>
  <li>types/openconfig-yang-types.yang (1): warning: <pre>Module openconfig-yang-types is missing a grouping suffixed with -top</pre></li>
  <li>vlan/openconfig-vlan-types.yang (1): warning: <pre>Module openconfig-vlan-types is missing a grouping suffixed with -top</pre></li>
  <li>wifi/types/openconfig-wifi-types.yang (1): warning: <pre>Module openconfig-wifi-types is missing a grouping suffixed with -top</pre></li>
</ul>
`,
	}, {
		name: "only warnings, but no warnings for output",
		in: `/workspace/release/yang/acl/openconfig-packet-match-types.yang:1: warning: Module openconfig-packet-match-types is missing a grouping suffixed with -top
/workspace/release/yang/openconfig-extensions.yang:49: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:158: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:169: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/types/openconfig-inet-types.yang:1: warning: Module openconfig-inet-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-types.yang:1: warning: Module openconfig-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-yang-types.yang:1: warning: Module openconfig-yang-types is missing a grouping suffixed with -top
`,
		inPass:       true,
		inNoWarnings: true,
		want: `Passed.
`,
	}, {
		name: "warnings and errors, but no warnings for output, and prioritizing errors",
		in: `/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "A" should be of the form UPPERCASE_WITH_UNDERSCORES: A
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "G" should be of the form UPPERCASE_WITH_UNDERSCORES: G
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "N" should be of the form UPPERCASE_WITH_UNDERSCORES: N
/workspace/release/yang/openconfig-extensions.yang:49: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:158: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/openconfig-extensions.yang:169: warning: RFC 6087: 4.3: statement "yin-element" is given with its default value "false"
/workspace/release/yang/types/openconfig-inet-types.yang:1: warning: Module openconfig-inet-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-types.yang:1: warning: Module openconfig-types is missing a grouping suffixed with -top
/workspace/release/yang/types/openconfig-yang-types.yang:1: warning: Module openconfig-yang-types is missing a grouping suffixed with -top
/workspace/release/yang/vlan/openconfig-vlan-types.yang:1: warning: Module openconfig-vlan-types is missing a grouping suffixed with -top
/workspace/release/yang/wifi/types/openconfig-wifi-types.yang:1: warning: Module openconfig-wifi-types is missing a grouping suffixed with -top
/workspace/release/yang/wifi/types/openconfig-wifi-types.yang:288: error: identity name "BETTER-CHANNEL" should be of the form UPPERCASE_WITH_UNDERSCORES: "BETTER-CHANNEL"
`,
		inPass:       false,
		inNoWarnings: true,
		want: `<ul>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "A" should be of the form UPPERCASE_WITH_UNDERSCORES: A</pre></li>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B</pre></li>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "G" should be of the form UPPERCASE_WITH_UNDERSCORES: G</pre></li>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "N" should be of the form UPPERCASE_WITH_UNDERSCORES: N</pre></li>
  <li>wifi/types/openconfig-wifi-types.yang (288): error: <pre>identity name "BETTER-CHANNEL" should be of the form UPPERCASE_WITH_UNDERSCORES: "BETTER-CHANNEL"</pre></li>
</ul>
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processAnyPyangOutput(tt.in, tt.inPass, tt.inNoWarnings)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(strings.Split(tt.want, "\n"), strings.Split(got, "\n")); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGetResult(t *testing.T) {
	modelRoot = "/workspace/release/yang"

	tests := []struct {
		name                 string
		inValidatorResultDir string
		inValidatorId        string
		wantPass             bool
		wantOut              string
		wantErrSubstr        string
	}{{
		name:                 "basic pyang pass",
		inValidatorResultDir: "testdata/oc-pyang",
		inValidatorId:        "oc-pyang",
		wantPass:             true,
		wantOut: `<details>
  <summary>:white_check_mark: acl</summary>
<details>
  <summary>:white_check_mark: openconfig-acl</summary>
Passed.
</details>
</details>
<details>
  <summary>:white_check_mark: optical-transport</summary>
<details>
  <summary>:white_check_mark: openconfig-optical-amplifier</summary>
Passed.
</details>
<details>
  <summary>:white_check_mark: openconfig-transport-line-protection</summary>
Passed.
<ul>
  <li>warning foo</li>
</ul>
</details>
</details>
`,
	}, {
		name:                 "pyang with an empty fail file",
		inValidatorResultDir: "testdata/oc-pyang-with-fail-file",
		inValidatorId:        "oc-pyang",
		wantPass:             false,
		wantOut: `Validator script failed -- infra bug?
Test failed with no stderr output.`,
	}, {
		name:                 "basic non-pyang pass",
		inValidatorResultDir: "testdata/oc-pyang",
		inValidatorId:        "goyang-ygot",
		wantPass:             true,
		wantOut: `<details>
  <summary>:white_check_mark: acl</summary>
<details>
  <summary>:white_check_mark: openconfig-acl</summary>
Passed.
</details>
</details>
<details>
  <summary>:white_check_mark: optical-transport</summary>
<details>
  <summary>:white_check_mark: openconfig-optical-amplifier</summary>
Passed.
</details>
<details>
  <summary>:white_check_mark: openconfig-transport-line-protection</summary>
Passed.
warning foo<br>
</details>
</details>
`,
	}, {
		name:                 "pyang with pass and fails",
		inValidatorResultDir: "testdata/pyang-with-invalid-files",
		inValidatorId:        "pyang",
		wantPass:             false,
		wantOut: `<details>
  <summary>:no_entry: acl</summary>
<details>
  <summary>:no_entry: openconfig-acl</summary>
<ul>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B</pre></li>
</ul>
</details>
</details>
<details>
  <summary>:no_entry: optical-transport</summary>
<details>
  <summary>:no_entry: openconfig-optical-amplifier</summary>
Failed.
</details>
<details>
  <summary>:white_check_mark: openconfig-transport-line-protection</summary>
Passed.
<ul>
  <li>warning foo</li>
</ul>
</details>
</details>
`,
	}, {
		name:                 "non-pyang with pass and fails",
		inValidatorResultDir: "testdata/pyang-with-invalid-files",
		inValidatorId:        "yanglint",
		wantPass:             false,
		wantOut: `<details>
  <summary>:no_entry: acl</summary>
<details>
  <summary>:no_entry: openconfig-acl</summary>
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B<br>
</details>
</details>
<details>
  <summary>:no_entry: optical-transport</summary>
<details>
  <summary>:no_entry: openconfig-optical-amplifier</summary>
Failed.
</details>
<details>
  <summary>:white_check_mark: openconfig-transport-line-protection</summary>
Passed.
warning foo<br>
</details>
</details>
`,
	}, {
		name:                 "non-per-model pass -- no fail file",
		inValidatorResultDir: "testdata/regexp-tests",
		inValidatorId:        "regexp",
		wantPass:             true,
		wantOut:              `Test passed.`,
	}, {
		name:                 "non-per-model fail -- empty fail file",
		inValidatorResultDir: "testdata/regexp-tests2",
		inValidatorId:        "regexp",
		wantPass:             false,
		wantOut:              `Test failed with no stderr output.`,
	}, {
		name:                 "non-per-model fail",
		inValidatorResultDir: "testdata/regexp-tests-fail",
		inValidatorId:        "regexp",
		wantPass:             false,
		wantOut:              "I failed\n",
	}, {
		name:                 "pyang script fail",
		inValidatorResultDir: "testdata/oc-pyang-script-fail",
		inValidatorId:        "oc-pyang",
		wantPass:             false,
		wantOut:              "Validator script failed -- infra bug?\nI failed\n",
	}, {
		name:                 "openconfig-version, revision version, and .spec.yml checks all pass",
		inValidatorResultDir: "testdata/misc-checks-pass",
		inValidatorId:        "misc-checks",
		wantPass:             true,
		wantOut: `<details>
  <summary>:white_check_mark: openconfig-version update check</summary>
4 file(s) correctly updated.
</details>
<details>
  <summary>:white_check_mark: .spec.yml build reachability check</summary>
8 files reached by build rules.
</details>
`,
	}, {
		name:                 "openconfig-version, revision version, and .spec.yml checks all fail",
		inValidatorResultDir: "testdata/misc-checks-fail",
		inValidatorId:        "misc-checks",
		wantPass:             false,
		wantOut: `<details>
  <summary>:no_entry: openconfig-version update check</summary>
  <li>changed-version-to-noversion.yang: openconfig-version was removed</li>
  <li>openconfig-acl.yang: file updated but PR version not updated: "1.2.2"</li>
</details>
<details>
  <summary>:no_entry: .spec.yml build reachability check</summary>
  <li>changed-noversion-to-unreached.yang: file not used by any .spec.yml build.</li>
  <li>changed-unreached-to-unreached.yang: file not used by any .spec.yml build.</li>
  <li>changed-version-to-unreached.yang: file not used by any .spec.yml build.</li>
  <li>unchanged-unreached.yang: file not used by any .spec.yml build.</li>
</details>
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotPass, err := getResult(tt.inValidatorId, tt.inValidatorResultDir)
			if err != nil {
				if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
					t.Fatalf("did not get expected error, %s", diff)
				}
				return
			}
			if gotPass != tt.wantPass {
				t.Errorf("gotPass %v, want %v", gotPass, tt.wantPass)
			}
			if diff := cmp.Diff(strings.Split(tt.wantOut, "\n"), strings.Split(gotOut, "\n")); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGetGistInfo(t *testing.T) {
	tests := []struct {
		name                 string
		inValidatorResultDir string
		inValidatorId        string
		inVersion            string
		wantDescription      string
		wantContent          string
		wantErrSubstr        string
	}{{
		name:                 "oc-pyang with output and latest-version.txt file",
		inValidatorResultDir: "testdata/oc-pyang",
		inValidatorId:        "oc-pyang",
		wantDescription:      "yanglint@SO 1.5.5",
		wantContent:          "foo\n",
	}, {
		name:                 "invalid validator name",
		inValidatorResultDir: "testdata/oc-pyang",
		inValidatorId:        "oc-pyin",
		wantErrSubstr:        `validator "oc-pyin" not found`,
	}, {
		name:                 "regexp with no output and no latest-version.txt file",
		inValidatorResultDir: "testdata/regexp-tests",
		inValidatorId:        "regexp",
		wantDescription:      "regexp tests",
		wantContent:          "No output",
	}, {
		name:                 "regexp with no output but with latest-version.txt file with no spaces in the version name",
		inValidatorResultDir: "testdata/regexp-tests2",
		inValidatorId:        "regexp",
		wantDescription:      "regexp-1.2",
		wantContent:          "No output",
	}, {
		name:                 "pyang with missing output file",
		inValidatorResultDir: "testdata/pyang-with-invalid-files",
		inValidatorId:        "pyang",
		wantErrSubstr:        "no such file",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDescription, gotContent, err := getGistHeading(tt.inValidatorId, tt.inVersion, tt.inValidatorResultDir)
			if err != nil {
				if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
					t.Fatalf("did not get expected error, %s", diff)
				}
				return
			}
			if gotDescription != tt.wantDescription {
				t.Errorf("gotDescription %q, want %q", gotDescription, tt.wantDescription)
			}
			if gotContent != tt.wantContent {
				t.Errorf("gotContent %v, want %v", gotContent, tt.wantContent)
			}
		})
	}
}
