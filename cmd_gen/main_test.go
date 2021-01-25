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
	"github.com/openconfig/models-ci/commonci"
)

// Fake LabelPoster for testing.
type postLabelRecorder struct {
	labels []string
}

func (p *postLabelRecorder) PostLabel(labelName, labelColor, owner, repo string, prNumber int) error {
	p.labels = append(p.labels, labelName)
	return nil
}

func TestGenOpenConfigValidatorScript(t *testing.T) {
	prNumber = 1
	basicModelMap, err := commonci.ParseOCModels("testdata")
	if err != nil {
		t.Fatalf("TestGenOpenConfigLinterScript: Failed to parse models for testing: %v", err)
	}

	tests := []struct {
		name                 string
		inValidatorName      string
		inModelMap           commonci.OpenConfigModelMap
		inDisabledModelPaths map[string]bool
		wantCmd              string
		wantSkipLabels       []string
		wantErr              bool
	}{{
		name:            "basic pyang",
		inModelMap:      basicModelMap,
		inValidatorName: "pyang",
		wantCmd: `#!/bin/bash
mkdir -p /workspace/results/pyang
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/pyang/acl==openconfig-acl==pass; then
  mv /workspace/results/pyang/acl==openconfig-acl==pass /workspace/results/pyang/acl==openconfig-acl==fail
fi &
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==pass /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==fail
fi &
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==pass /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==fail
fi &
wait
`,
	}, {
		name:                 "basic pyang with model to be skipped",
		inModelMap:           basicModelMap,
		inValidatorName:      "pyang",
		inDisabledModelPaths: map[string]bool{"acl": true, "dne": true},
		wantSkipLabels:       []string{"skipped: acl"},
		wantCmd: `#!/bin/bash
mkdir -p /workspace/results/pyang
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==pass /workspace/results/pyang/optical-transport==openconfig-optical-amplifier==fail
fi &
if ! $@ -p testdata -p /workspace/third_party/ietf testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==pass /workspace/results/pyang/optical-transport==openconfig-transport-line-protection==fail
fi &
wait
`,
	}, {
		name:            "basic oc-pyang",
		inModelMap:      basicModelMap,
		inValidatorName: "oc-pyang",
		wantCmd: `#!/bin/bash
mkdir -p /workspace/results/oc-pyang
if ! $@ -p testdata -p /workspace/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/oc-pyang/acl==openconfig-acl==pass; then
  mv /workspace/results/oc-pyang/acl==openconfig-acl==pass /workspace/results/oc-pyang/acl==openconfig-acl==fail
fi &
if ! $@ -p testdata -p /workspace/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/oc-pyang/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/oc-pyang/optical-transport==openconfig-optical-amplifier==pass /workspace/results/oc-pyang/optical-transport==openconfig-optical-amplifier==fail
fi &
if ! $@ -p testdata -p /workspace/third_party/ietf --openconfig --ignore-error=OC_RELATIVE_PATH testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/oc-pyang/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/oc-pyang/optical-transport==openconfig-transport-line-protection==pass /workspace/results/oc-pyang/optical-transport==openconfig-transport-line-protection==fail
fi &
wait
`,
	}, {
		name:            "basic pyangbind",
		inModelMap:      basicModelMap,
		inValidatorName: "pyangbind",
		wantCmd: `#!/bin/bash
mkdir -p /workspace/results/pyangbind
if ! $@ -p testdata -p /workspace/third_party/ietf -f pybind -o acl.openconfig-acl.binding.py testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/pyangbind/acl==openconfig-acl==pass; then
  mv /workspace/results/pyangbind/acl==openconfig-acl==pass /workspace/results/pyangbind/acl==openconfig-acl==fail
fi &
if ! $@ -p testdata -p /workspace/third_party/ietf -f pybind -o optical-transport.openconfig-optical-amplifier.binding.py testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/pyangbind/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/pyangbind/optical-transport==openconfig-optical-amplifier==pass /workspace/results/pyangbind/optical-transport==openconfig-optical-amplifier==fail
fi &
if ! $@ -p testdata -p /workspace/third_party/ietf -f pybind -o optical-transport.openconfig-transport-line-protection.binding.py testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/pyangbind/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/pyangbind/optical-transport==openconfig-transport-line-protection==pass /workspace/results/pyangbind/optical-transport==openconfig-transport-line-protection==fail
fi &
wait
`,
	}, {
		name:            "basic goyang-ygot",
		inModelMap:      basicModelMap,
		inValidatorName: "goyang-ygot",
		wantCmd: `#!/bin/bash
mkdir -p /workspace/results/goyang-ygot
if ! /go/bin/generator \
-path=testdata,/workspace/third_party/ietf \
-output_file=/workspace/results/goyang-ygot/acl.openconfig-acl.oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &> /workspace/results/goyang-ygot/acl==openconfig-acl==pass; then
  mv /workspace/results/goyang-ygot/acl==openconfig-acl==pass /workspace/results/goyang-ygot/acl==openconfig-acl==fail
fi &
if ! /go/bin/generator \
-path=testdata,/workspace/third_party/ietf \
-output_file=/workspace/results/goyang-ygot/optical-transport.openconfig-optical-amplifier.oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
testdata/optical-transport/openconfig-optical-amplifier.yang &> /workspace/results/goyang-ygot/optical-transport==openconfig-optical-amplifier==pass; then
  mv /workspace/results/goyang-ygot/optical-transport==openconfig-optical-amplifier==pass /workspace/results/goyang-ygot/optical-transport==openconfig-optical-amplifier==fail
fi &
if ! /go/bin/generator \
-path=testdata,/workspace/third_party/ietf \
-output_file=/workspace/results/goyang-ygot/optical-transport.openconfig-transport-line-protection.oc.go \
-package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true \
-exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters \
-generate_leaf_getters -generate_delete -annotations \
testdata/optical-transport/openconfig-transport-line-protection.yang &> /workspace/results/goyang-ygot/optical-transport==openconfig-transport-line-protection==pass; then
  mv /workspace/results/goyang-ygot/optical-transport==openconfig-transport-line-protection==pass /workspace/results/goyang-ygot/optical-transport==openconfig-transport-line-protection==fail
fi &
wait
`,
	}, {
		name:            "basic yanglint",
		inModelMap:      basicModelMap,
		inValidatorName: "yanglint",
		wantCmd: `#!/bin/bash
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
wait
`,
	}, {
		name:            "basic confd",
		inModelMap:      basicModelMap,
		inValidatorName: "confd",
		wantCmd: `#!/bin/bash
mkdir -p /workspace/results/confd
status=0
$1 -c --yangpath $2 testdata/acl/openconfig-acl.yang &>> /workspace/results/confd/acl==openconfig-acl==pass || status=1
$1 -c --yangpath $2 testdata/acl/openconfig-acl-evil-twin.yang &>> /workspace/results/confd/acl==openconfig-acl==pass || status=1
if [[ $status -eq "1" ]]; then
  mv /workspace/results/confd/acl==openconfig-acl==pass /workspace/results/confd/acl==openconfig-acl==fail
fi
status=0
$1 -c --yangpath $2 testdata/optical-transport/openconfig-optical-amplifier.yang &>> /workspace/results/confd/optical-transport==openconfig-optical-amplifier==pass || status=1
if [[ $status -eq "1" ]]; then
  mv /workspace/results/confd/optical-transport==openconfig-optical-amplifier==pass /workspace/results/confd/optical-transport==openconfig-optical-amplifier==fail
fi
status=0
$1 -c --yangpath $2 testdata/optical-transport/openconfig-transport-line-protection.yang &>> /workspace/results/confd/optical-transport==openconfig-transport-line-protection==pass || status=1
if [[ $status -eq "1" ]]; then
  mv /workspace/results/confd/optical-transport==openconfig-transport-line-protection==pass /workspace/results/confd/optical-transport==openconfig-transport-line-protection==fail
fi
wait
`,
	}, {
		name:            "basic misc-checks",
		inModelMap:      basicModelMap,
		inValidatorName: "misc-checks",
		wantCmd: `#!/bin/bash
mkdir -p /workspace/results/misc-checks
if ! /go/bin/ocversion -p testdata,/workspace/third_party/ietf testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang > /workspace/results/misc-checks/acl.openconfig-acl.pr-file-parse-log; then
  >&2 echo "parse of acl.openconfig-acl reported non-zero status."
fi
if ! /go/bin/ocversion -p testdata,/workspace/third_party/ietf testdata/optical-transport/openconfig-optical-amplifier.yang > /workspace/results/misc-checks/optical-transport.openconfig-optical-amplifier.pr-file-parse-log; then
  >&2 echo "parse of optical-transport.openconfig-optical-amplifier reported non-zero status."
fi
if ! /go/bin/ocversion -p testdata,/workspace/third_party/ietf testdata/optical-transport/openconfig-transport-line-connectivity.yang testdata/optical-transport/openconfig-wavelength-router.yang > /workspace/results/misc-checks/optical-transport.openconfig-wavelength-router.pr-file-parse-log; then
  >&2 echo "parse of optical-transport.openconfig-wavelength-router reported non-zero status."
fi
if ! /go/bin/ocversion -p testdata,/workspace/third_party/ietf testdata/optical-transport/openconfig-transport-line-protection.yang > /workspace/results/misc-checks/optical-transport.openconfig-transport-line-protection.pr-file-parse-log; then
  >&2 echo "parse of optical-transport.openconfig-transport-line-protection reported non-zero status."
fi
if ! /go/bin/ocversion -p testdata,/workspace/third_party/ietf testdata/optical-transport/openconfig-optical-attenuator.yang > /workspace/results/misc-checks/optical-transport.openconfig-optical-attenuator.pr-file-parse-log; then
  >&2 echo "parse of optical-transport.openconfig-optical-attenuator reported non-zero status."
fi
wait
`,
	}, {
		name:            "unrecognized validatorID",
		inModelMap:      basicModelMap,
		inValidatorName: "foo",
		wantErr:         true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labelRecorder := &postLabelRecorder{}
			disabledModelPaths = tt.inDisabledModelPaths

			got, err := genOpenConfigValidatorScript(labelRecorder, tt.inValidatorName, "", tt.inModelMap)
			if got := err != nil; got != tt.wantErr {
				t.Fatalf("got error %v,	wantErr: %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(strings.Split(tt.wantCmd, "\n"), strings.Split(got, "\n")); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.wantSkipLabels, labelRecorder.labels); diff != "" {
				t.Errorf("skipped models (-want, +got):\n%s", diff)
			}
		})
	}
}
