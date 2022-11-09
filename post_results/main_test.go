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
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
)

func TestProcessStandardOutput(t *testing.T) {
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
	}, {
		name: "ConfD sample output",
		in: `/workspace/release/yang/platform/openconfig-platform-port.yang:139: warning: the node is config, but refers to a non-config node 'type' defined at /workspace/release/yang/platform/openconfig-platform.yang:302
/workspace/release/yang/platform/openconfig-platform-port.yang:139: warning: the node is config, but refers to a non-config node 'type' defined at /workspace/release/yang/platform/openconfig-platform.yang:302
/workspace/release/yang/platform/openconfig-platform-transceiver.yang:557: warning: the node is config, but refers to a non-config node 'type' defined at /workspace/release/yang/platform/openconfig-platform.yang:302
`,
		inPass:       true,
		inNoWarnings: false,
		want: `Passed.
<ul>
  <li>platform/openconfig-platform-port.yang (139): warning: <pre>the node is config, but refers to a non-config node 'type' defined at /workspace/release/yang/platform/openconfig-platform.yang:302</pre></li>
  <li>platform/openconfig-platform-port.yang (139): warning: <pre>the node is config, but refers to a non-config node 'type' defined at /workspace/release/yang/platform/openconfig-platform.yang:302</pre></li>
  <li>platform/openconfig-platform-transceiver.yang (557): warning: <pre>the node is config, but refers to a non-config node 'type' defined at /workspace/release/yang/platform/openconfig-platform.yang:302</pre></li>
</ul>
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processStandardOutput(tt.in, tt.inPass, tt.inNoWarnings)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(strings.Split(tt.want, "\n"), strings.Split(got, "\n")); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}

func TestCheckSemverIncrease(t *testing.T) {
	tests := []struct {
		desc          string
		inOldVersion  string
		inNewVersion  string
		wantErrSubstr string
	}{{
		desc:         "single increase",
		inOldVersion: "1.0.0",
		inNewVersion: "1.0.1",
	}, {
		desc:          "no change",
		inOldVersion:  "1.0.1",
		inNewVersion:  "1.0.1",
		wantErrSubstr: "file updated but test-version string not updated",
	}, {
		desc:          "decrease",
		inOldVersion:  "1.0.1",
		inNewVersion:  "1.0.0",
		wantErrSubstr: "new semantic version not valid",
	}, {
		desc:          "invalid old version",
		inOldVersion:  "1.0.*",
		inNewVersion:  "1.0.0",
		wantErrSubstr: "base branch version string unparseable",
	}, {
		desc:          "invalid new version",
		inOldVersion:  "1.0.0",
		inNewVersion:  "1.0.*",
		wantErrSubstr: "invalid version string",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := checkSemverIncrease(tt.inOldVersion, tt.inNewVersion, "test-version")
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
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
		wantCondensedOut     string
		wantCondensedOutSame bool
		wantErrSubstr        string
	}{{
		name:                 "basic pyang pass",
		inValidatorResultDir: "testdata/oc-pyang",
		inValidatorId:        "oc-pyang",
		wantPass:             true,
		wantOut: `<details>
  <summary>&#x2705;&nbsp; acl</summary>
<details>
  <summary>&#x2705;&nbsp; openconfig-acl</summary>
&#x1F4B2;&nbsp; bash command
<pre>foo command
$WORKSPACE/foo/bar
$GOPATH/src/github.com/openconfig/oc-pyang/openconfig_pyang/plugins
$GOPATH/src/github.com/robshakir/pyangbind/pyangbind/plugin
</pre>
Passed.
</details>
</details>
<details>
  <summary>&#x2705;&nbsp; optical-transport</summary>
<details>
  <summary>&#x2705;&nbsp; openconfig-optical-amplifier</summary>
Passed.
</details>
<details>
  <summary>&#x2705;&nbsp; openconfig-transport-line-protection</summary>
Passed.
<ul>
  <pre>warning foo</pre>
</ul>
</details>
</details>
`,
		wantCondensedOut: `All passed.
`,
	}, {
		name:                 "pyang with an empty fail file",
		inValidatorResultDir: "testdata/oc-pyang-with-fail-file",
		inValidatorId:        "oc-pyang",
		wantPass:             false,
		wantOut: `Validator script failed -- infra bug?
Test failed with no stderr output.`,
		wantCondensedOutSame: true,
	}, {
		name:                 "basic non-pyang pass",
		inValidatorResultDir: "testdata/oc-pyang",
		inValidatorId:        "goyang-ygot",
		wantPass:             true,
		wantOut: `<details>
  <summary>&#x2705;&nbsp; acl</summary>
<details>
  <summary>&#x2705;&nbsp; openconfig-acl</summary>
&#x1F4B2;&nbsp; bash command
<pre>foo command
$WORKSPACE/foo/bar
$GOPATH/src/github.com/openconfig/oc-pyang/openconfig_pyang/plugins
$GOPATH/src/github.com/robshakir/pyangbind/pyangbind/plugin
</pre>
Passed.
</details>
</details>
<details>
  <summary>&#x2705;&nbsp; optical-transport</summary>
<details>
  <summary>&#x2705;&nbsp; openconfig-optical-amplifier</summary>
Passed.
</details>
<details>
  <summary>&#x2705;&nbsp; openconfig-transport-line-protection</summary>
Passed.
warning foo<br>
</details>
</details>
`,
		wantCondensedOut: `All passed.
`,
	}, {
		name:                 "pyang with pass and fails",
		inValidatorResultDir: "testdata/pyang-with-invalid-files",
		inValidatorId:        "pyang",
		wantPass:             false,
		wantOut: `<details>
  <summary>&#x26D4;&nbsp; acl</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-acl</summary>
<ul>
  <li>acl/openconfig-acl.yang (845): error: <pre>grouping "acl-state" not found in module "openconfig-acl"</pre></li>
</ul>
</details>
</details>
<details>
  <summary>&#x26D4;&nbsp; optical-transport</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-optical-amplifier</summary>
Failed.
</details>
<details>
  <summary>&#x2705;&nbsp; openconfig-transport-line-protection</summary>
Passed.
<ul>
  <pre>warning foo</pre>
</ul>
</details>
</details>
`,
		wantCondensedOut: `<details>
  <summary>&#x26D4;&nbsp; acl</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-acl</summary>
<ul>
  <li>acl/openconfig-acl.yang (845): error: <pre>grouping "acl-state" not found in module "openconfig-acl"</pre></li>
</ul>
</details>
</details>
<details>
  <summary>&#x26D4;&nbsp; optical-transport</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-optical-amplifier</summary>
Failed.
</details>
</details>
`,
	}, {
		name:                 "confd with pass and fails",
		inValidatorResultDir: "testdata/confd-with-invalid-files",
		inValidatorId:        "confd",
		wantPass:             false,
		wantOut: `<details>
  <summary>&#x26D4;&nbsp; acl</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-acl</summary>
<ul>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B</pre></li>
</ul>
</details>
</details>
<details>
  <summary>&#x26D4;&nbsp; optical-transport</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-optical-amplifier</summary>
Failed.
</details>
<details>
  <summary>&#x2705;&nbsp; openconfig-transport-line-protection</summary>
Passed.
<ul>
  <li>warning foo</li>
</ul>
</details>
</details>
`,
		wantCondensedOut: `<details>
  <summary>&#x26D4;&nbsp; acl</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-acl</summary>
<ul>
  <li>wifi/mac/openconfig-wifi-mac.yang (1244): error: <pre>enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B</pre></li>
</ul>
</details>
</details>
<details>
  <summary>&#x26D4;&nbsp; optical-transport</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-optical-amplifier</summary>
Failed.
</details>
</details>
`,
	}, {
		name:                 "non-pyang with pass and fails",
		inValidatorResultDir: "testdata/confd-with-invalid-files",
		inValidatorId:        "yanglint",
		wantPass:             false,
		wantOut: `<details>
  <summary>&#x26D4;&nbsp; acl</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-acl</summary>
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B<br>
</details>
</details>
<details>
  <summary>&#x26D4;&nbsp; optical-transport</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-optical-amplifier</summary>
Failed.
</details>
<details>
  <summary>&#x2705;&nbsp; openconfig-transport-line-protection</summary>
Passed.
warning foo<br>
</details>
</details>
`,
		wantCondensedOut: `<details>
  <summary>&#x26D4;&nbsp; acl</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-acl</summary>
/workspace/release/yang/wifi/mac/openconfig-wifi-mac.yang:1244: error: enum value "B" should be of the form UPPERCASE_WITH_UNDERSCORES: B<br>
</details>
</details>
<details>
  <summary>&#x26D4;&nbsp; optical-transport</summary>
<details>
  <summary>&#x26D4;&nbsp; openconfig-optical-amplifier</summary>
Failed.
</details>
</details>
`,
	}, {
		name:                 "non-per-model pass -- no fail file",
		inValidatorResultDir: "testdata/regexp-tests",
		inValidatorId:        "regexp",
		wantPass:             true,
		wantOut:              `Test passed.`,
		wantCondensedOutSame: true,
	}, {
		name:                 "non-per-model fail -- empty fail file",
		inValidatorResultDir: "testdata/regexp-tests2",
		inValidatorId:        "regexp",
		wantPass:             false,
		wantOut:              `Test failed with no stderr output.`,
		wantCondensedOutSame: true,
	}, {
		name:                 "non-per-model fail",
		inValidatorResultDir: "testdata/regexp-tests-fail",
		inValidatorId:        "regexp",
		wantPass:             false,
		wantOut:              "I failed\n",
		wantCondensedOutSame: true,
	}, {
		name:                 "pyang script fail",
		inValidatorResultDir: "testdata/oc-pyang-script-fail",
		inValidatorId:        "oc-pyang",
		wantPass:             false,
		wantOut:              "Validator script failed -- infra bug?\nI failed\n",
		wantCondensedOutSame: true,
	}, {
		name:                 "openconfig-version, revision version, and .spec.yml checks all pass",
		inValidatorResultDir: "testdata/misc-checks-pass",
		inValidatorId:        "misc-checks",
		wantPass:             true,
		wantOut: `<details>
  <summary>&#x2705;&nbsp; openconfig-version update check</summary>
7 file(s) correctly updated.
</details>
<details>
  <summary>&#x2705;&nbsp; .spec.yml build reachability check</summary>
9 files reached by build rules.
</details>
<details>
  <summary>&#x2705;&nbsp; submodule versions must match the belonging module's version</summary>
5 module/submodule file groups have matching versions</details>
`,
		wantCondensedOutSame: true,
	}, {
		name:                 "openconfig-version, revision version, and .spec.yml checks all fail",
		inValidatorResultDir: "testdata/misc-checks-fail",
		inValidatorId:        "misc-checks",
		wantPass:             false,
		wantOut: `<details>
  <summary>&#x26D4;&nbsp; openconfig-version update check</summary>
  <li>changed-version-to-noversion.yang: openconfig-version was removed</li>
  <li>openconfig-acl.yang: file updated but openconfig-version string not updated: "1.2.2"</li>
  <li>openconfig-mpls.yang: new semantic version not valid, old version: "2.3.4", new version: "2.2.5"</li>
</details>
<details>
  <summary>&#x26D4;&nbsp; .spec.yml build reachability check</summary>
  <li>changed-noversion-to-unreached.yang: file not used by any .spec.yml build.</li>
  <li>changed-unreached-to-unreached.yang: file not used by any .spec.yml build.</li>
  <li>changed-version-to-unreached.yang: file not used by any .spec.yml build.</li>
  <li>unchanged-unreached.yang: file not used by any .spec.yml build.</li>
</details>
<details>
  <summary>&#x26D4;&nbsp; submodule versions must match the belonging module's version</summary>
  <li>module set openconfig-mpls is at <b>2.3.4</b> (openconfig-mpls-submodule.yang), non-matching files: <b>openconfig-mpls-submodule2.yang</b> (2.3.2), <b>openconfig-mpls.yang</b> (2.2.5)</li>
</details>
`,
		wantCondensedOutSame: true,
	}}

	for _, tt := range tests {
		for _, condensed := range []bool{false, true} {
			t.Run(fmt.Sprintf(tt.name+"@condensed=%v", condensed), func(t *testing.T) {
				gotOut, gotPass, err := getResult(tt.inValidatorId, tt.inValidatorResultDir, condensed)
				if err != nil {
					if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
						t.Fatalf("did not get expected error, %s", diff)
					}
					return
				}
				if gotPass != tt.wantPass {
					t.Errorf("gotPass %v, want %v", gotPass, tt.wantPass)
				}
				wantOut := tt.wantOut
				if condensed && !tt.wantCondensedOutSame {
					wantOut = tt.wantCondensedOut
				}
				if diff := cmp.Diff(strings.Split(wantOut, "\n"), strings.Split(gotOut, "\n")); diff != "" {
					t.Errorf("(-want, +got):\n%s", diff)
				}
			})
		}
	}
}

func TestGetGistHeading(t *testing.T) {
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

func TestWriteBadgeUploadCmdFile(t *testing.T) {
	repoSlug = "openconfig/repo"
	tests := []struct {
		name                 string
		inValidatorDesc      string
		inValidatorUniqueStr string
		inVersion            string
		inPass               bool
		inResultsDir         string
		wantFileContent      string
	}{{
		name:                 "pass",
		inValidatorDesc:      "pyang@1.2.3",
		inValidatorUniqueStr: "pyang@latest",
		inPass:               true,
		inResultsDir:         "results-dir",
		wantFileContent: `REMOTE_PATH_PFX=gs://openconfig/compatibility-badges/openconfig-repo:
RESULTSDIR=results-dir
upload-public-file() {
	gsutil cp $RESULTSDIR/$1 "$REMOTE_PATH_PFX"$1
	gsutil acl ch -u AllUsers:R "$REMOTE_PATH_PFX"$1
	gsutil setmeta -h "Cache-Control:no-cache" "$REMOTE_PATH_PFX"$1
}
badge "pass" "pyang@1.2.3" :brightgreen > $RESULTSDIR/pyang@latest.svg
upload-public-file pyang@latest.svg
upload-public-file pyang@latest.html
`,
	}, {
		name:                 "fail",
		inValidatorDesc:      "pyang@2.3.4",
		inValidatorUniqueStr: "pyang",
		inPass:               false,
		inResultsDir:         "results-directory",
		wantFileContent: `REMOTE_PATH_PFX=gs://openconfig/compatibility-badges/openconfig-repo:
RESULTSDIR=results-directory
upload-public-file() {
	gsutil cp $RESULTSDIR/$1 "$REMOTE_PATH_PFX"$1
	gsutil acl ch -u AllUsers:R "$REMOTE_PATH_PFX"$1
	gsutil setmeta -h "Cache-Control:no-cache" "$REMOTE_PATH_PFX"$1
}
badge "fail" "pyang@2.3.4" :red > $RESULTSDIR/pyang.svg
upload-public-file pyang.svg
upload-public-file pyang.html
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WriteBadgeUploadCmdFile(tt.inValidatorDesc, tt.inValidatorUniqueStr, tt.inPass, tt.inResultsDir)
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.wantFileContent {
				t.Errorf("gotFileContent:\n%v\nwant:\n%v", got, tt.wantFileContent)
			}
		})
	}
}
