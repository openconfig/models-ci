package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProcessOcPyangOutput(t *testing.T) {
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

func TestParseResultsMd(t *testing.T) {
	tests := []struct {
		name                 string
		inValidatorResultDir string
		validatorId          string
		wantPass             bool
		wantMd               string
	}{{
		name:                 "basic pyang pass",
		inValidatorResultDir: "testdata/oc-pyang",
		validatorId:          "oc-pyang",
		wantPass:             true,
		wantMd: `<details>
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
		name:                 "basic non-pyang pass",
		inValidatorResultDir: "testdata/oc-pyang",
		validatorId:          "goyang-ygot",
		wantPass:             true,
		wantMd: `<details>
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
		// TODO(wenbli): Add more tests to cover all cases.
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatorId = tt.validatorId
			gotMd, gotPass, err := parseResultsMd(tt.inValidatorResultDir)
			if err != nil {
				t.Fatal(err)
			}
			if gotPass != tt.wantPass {
				t.Errorf("gotPass %v, want %v", gotPass, tt.wantPass)
			}
			if diff := cmp.Diff(strings.Split(tt.wantMd, "\n"), strings.Split(gotMd, "\n")); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}
