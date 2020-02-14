package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/models-ci/commonci"
)

var (
	basicModelMap = OpenConfigModelMap{
		ModelRoot: "testdata",
		ModelInfoMap: map[string][]ModelInfo{
			"acl": []ModelInfo{{
				Name: "openconfig-acl",
				DocFiles: []string{
					"yang/acl/openconfig-packet-match-types.yang",
					"yang/acl/openconfig-acl.yang",
				},
				BuildFiles: []string{
					"testdata/acl/openconfig-acl.yang",
					"testdata/acl/openconfig-acl-evil-twin.yang",
				},
				RunCi: true,
			}},
			"optical-transport": []ModelInfo{{
				Name: "openconfig-terminal-device",
				DocFiles: []string{
					"yang/optical-transport/openconfig-transport-types.yang",
					"yang/platform/openconfig-platform-types.yang",
					"yang/optical-transport/openconfig-terminal-device.yang",
					"yang/platform/openconfig-platform-transceiver.yang",
				},
				RunCi: true,
			}, {
				Name: "openconfig-optical-amplifier",
				BuildFiles: []string{
					"testdata/optical-transport/openconfig-optical-amplifier.yang",
				},
				RunCi: true,
			}, {
				Name: "openconfig-wavelength-router",
				DocFiles: []string{
					"yang/optical-transport/openconfig-transport-types.yang",
					"yang/optical-transport/openconfig-transport-line-common.yang",
					"yang/optical-transport/openconfig-wavelength-router.yang",
					"yang/optical-transport/openconfig-channel-monitor.yang",
					"yang/optical-transport/openconfig-transport-line-connectivity.yang",
				},
				BuildFiles: []string{
					"testdata/optical-transport/openconfig-transport-line-connectivity.yang",
					"testdata/optical-transport/openconfig-wavelength-router.yang",
				},
				RunCi: false,
			}, {
				Name: "openconfig-transport-line-protection",
				DocFiles: []string{
					"yang/platform/openconfig-platform-types.yang",
					"yang/optical-transport/openconfig-transport-line-protection.yang",
					"yang/platform/openconfig-platform.yang",
				},
				BuildFiles: []string{
					"testdata/optical-transport/openconfig-transport-line-protection.yang",
				},
				RunCi: true,
			}, {
				Name: "openconfig-optical-attenuator",
				DocFiles: []string{
					"yang/optical-transport/openconfig-optical-attenuator.yang",
				},
				BuildFiles: []string{
					"testdata/optical-transport/openconfig-optical-attenuator.yang",
				},
				RunCi: false,
			}},
		},
	}
)

func TestParseModels(t *testing.T) {
	tests := []struct {
		name        string
		inModelRoot string
		want        OpenConfigModelMap
	}{{
		name:        "basic",
		inModelRoot: "testdata",
		want:        basicModelMap,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseModels(tt.inModelRoot)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}

func TestOpenConfigLinterCmd(t *testing.T) {
	tests := []struct {
		name            string
		inValidatorName string
		in              OpenConfigModelMap
		want            string
	}{{
		name:            "basic pyang",
		in:              basicModelMap,
		inValidatorName: "pyang",
		want: `#!/bin/bash
mkdir -p /workspace/results/pyang
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/pyang/acl==openconfig-acl==pass; then
  mv /workspace/results/pyang/acl==openconfig-acl==pass /workspace/results/pyang/acl==openconfig-acl==fail
fi
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==pass /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==fail
fi
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==pass /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==fail
fi
`,
	}, {
		name:            "basic oc-pyang",
		in:              basicModelMap,
		inValidatorName: "oc-pyang",
		want: `#!/bin/bash
mkdir -p /workspace/results/oc-pyang
if ! $@ -p testdata -p /workspace/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/oc-pyang/acl==openconfig-acl==pass; then
  mv /workspace/results/oc-pyang/acl==openconfig-acl==pass /workspace/results/oc-pyang/acl==openconfig-acl==fail
fi
if ! $@ -p testdata -p /workspace/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/oc-pyang/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/oc-pyang/optical-transport==openconfig-optical-amplifier==pass /workspace/results/oc-pyang/optical-transport==openconfig-optical-amplifier==fail
fi
if ! $@ -p testdata -p /workspace/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/oc-pyang/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/oc-pyang/optical-transport==openconfig-transport-line-protection==pass /workspace/results/oc-pyang/optical-transport==openconfig-transport-line-protection==fail
fi
`,
	}, {
		name:            "basic pyangbind",
		in:              basicModelMap,
		inValidatorName: "pyangbind",
		want: `#!/bin/bash
mkdir -p /workspace/results/pyangbind
if ! $@ -p testdata -p /workspace/third_party/ietf -f pybind -o binding.py testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/pyangbind/acl==openconfig-acl==pass; then
  mv /workspace/results/pyangbind/acl==openconfig-acl==pass /workspace/results/pyangbind/acl==openconfig-acl==fail
fi
if ! $@ -p testdata -p /workspace/third_party/ietf -f pybind -o binding.py testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/pyangbind/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/pyangbind/optical-transport==openconfig-optical-amplifier==pass /workspace/results/pyangbind/optical-transport==openconfig-optical-amplifier==fail
fi
if ! $@ -p testdata -p /workspace/third_party/ietf -f pybind -o binding.py testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/pyangbind/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/pyangbind/optical-transport==openconfig-transport-line-protection==pass /workspace/results/pyangbind/optical-transport==openconfig-transport-line-protection==fail
fi
`,
	}, {
		name:            "basic goyang-ygot",
		in:              basicModelMap,
		inValidatorName: "goyang-ygot",
		want: `#!/bin/bash
mkdir -p /workspace/results/goyang-ygot
if ! go run /go/src/github.com/openconfig/ygot/generator/generator.go \
-path=testdata,/workspace/third_party/ietf \
-output_file=/go/src/github.com/openconfig/ygot/exampleoc/oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/goyang-ygot/acl==openconfig-acl==pass; then
  mv /workspace/results/goyang-ygot/acl==openconfig-acl==pass /workspace/results/goyang-ygot/acl==openconfig-acl==fail
fi
if ! go run /go/src/github.com/openconfig/ygot/generator/generator.go \
-path=testdata,/workspace/third_party/ietf \
-output_file=/go/src/github.com/openconfig/ygot/exampleoc/oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/goyang-ygot/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/goyang-ygot/optical-transport==openconfig-optical-amplifier==pass /workspace/results/goyang-ygot/optical-transport==openconfig-optical-amplifier==fail
fi
if ! go run /go/src/github.com/openconfig/ygot/generator/generator.go \
-path=testdata,/workspace/third_party/ietf \
-output_file=/go/src/github.com/openconfig/ygot/exampleoc/oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/goyang-ygot/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/goyang-ygot/optical-transport==openconfig-transport-line-protection==pass /workspace/results/goyang-ygot/optical-transport==openconfig-transport-line-protection==fail
fi
`,
	}, {
		name:            "basic yanglint",
		in:              basicModelMap,
		inValidatorName: "yanglint",
		want: `#!/bin/bash
mkdir -p /workspace/results/yanglint
if ! yanglint -p testdata -p /workspace/third_party/ietf testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/yanglint/acl==openconfig-acl==pass; then
  mv /workspace/results/yanglint/acl==openconfig-acl==pass /workspace/results/yanglint/acl==openconfig-acl==fail
fi
if ! yanglint -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/yanglint/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/yanglint/optical-transport==openconfig-optical-amplifier==pass /workspace/results/yanglint/optical-transport==openconfig-optical-amplifier==fail
fi
if ! yanglint -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/yanglint/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/yanglint/optical-transport==openconfig-transport-line-protection==pass /workspace/results/yanglint/optical-transport==openconfig-transport-line-protection==fail
fi
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := genOpenConfigLinterCmd(nil, tt.inValidatorName, filepath.Join(commonci.ResultsDir, tt.inValidatorName), tt.in)
			if diff := cmp.Diff(strings.Split(tt.want, "\n"), strings.Split(got, "\n")); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}
