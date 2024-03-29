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
workdir=/workspace/results/pyang
mkdir -p "$workdir"
PYANG_MSG_TEMPLATE='messages:{{path:"{file}" line:{line} code:"{code}" type:"{type}" level:{level} message:'"'{msg}'}}"
cmd="$@"
options=(
  -p testdata
  -p /workspace/third_party/ietf
)
script_options=(
  --msg-template "$PYANG_MSG_TEMPLATE"
)
function run-dir() {
  declare prefix="$workdir"/"$1"=="$2"==
  shift 2
  echo pyang -W error "${options[@]}" "$@" > ${prefix}cmd
  if ! $($cmd -W error "${options[@]}" "${script_options[@]}" "$@" &> ${prefix}pass); then
    mv ${prefix}pass ${prefix}fail
  fi
}
run-dir "acl" "openconfig-acl" testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &
run-dir "optical-transport" "openconfig-optical-amplifier" testdata/optical-transport/openconfig-optical-amplifier.yang &
run-dir "optical-transport" "openconfig-transport-line-protection" testdata/optical-transport/openconfig-transport-line-protection.yang &
wait
`,
	}, {
		name:                 "basic pyang with model to be skipped",
		inModelMap:           basicModelMap,
		inValidatorName:      "pyang",
		inDisabledModelPaths: map[string]bool{"acl": true, "dne": true},
		wantSkipLabels:       []string{"skipped: acl"},
		wantCmd: `#!/bin/bash
workdir=/workspace/results/pyang
mkdir -p "$workdir"
PYANG_MSG_TEMPLATE='messages:{{path:"{file}" line:{line} code:"{code}" type:"{type}" level:{level} message:'"'{msg}'}}"
cmd="$@"
options=(
  -p testdata
  -p /workspace/third_party/ietf
)
script_options=(
  --msg-template "$PYANG_MSG_TEMPLATE"
)
function run-dir() {
  declare prefix="$workdir"/"$1"=="$2"==
  shift 2
  echo pyang -W error "${options[@]}" "$@" > ${prefix}cmd
  if ! $($cmd -W error "${options[@]}" "${script_options[@]}" "$@" &> ${prefix}pass); then
    mv ${prefix}pass ${prefix}fail
  fi
}
run-dir "optical-transport" "openconfig-optical-amplifier" testdata/optical-transport/openconfig-optical-amplifier.yang &
run-dir "optical-transport" "openconfig-transport-line-protection" testdata/optical-transport/openconfig-transport-line-protection.yang &
wait
`,
	}, {
		name:            "basic oc-pyang",
		inModelMap:      basicModelMap,
		inValidatorName: "oc-pyang",
		wantCmd: `#!/bin/bash
workdir=/workspace/results/oc-pyang
mkdir -p "$workdir"
PYANG_MSG_TEMPLATE='messages:{{path:"{file}" line:{line} code:"{code}" type:"{type}" level:{level} message:'"'{msg}'}}"
cmd="$@"
options=(
  --openconfig
  --ignore-error=OC_RELATIVE_PATH
  -p testdata
  -p /workspace/third_party/ietf
)
script_options=(
  --msg-template "$PYANG_MSG_TEMPLATE"
)
function run-dir() {
  declare prefix="$workdir"/"$1"=="$2"==
  local cmd_display_options=( --plugindir '$OCPYANG_PLUGIN_DIR' "${options[@]}" )
  local options=( --plugindir "$OCPYANG_PLUGIN_DIR" "${options[@]}" )
  shift 2
  echo pyang "${cmd_display_options[@]}" "$@" > ${prefix}cmd
  if ! $($cmd "${options[@]}" "${script_options[@]}" "$@" &> ${prefix}pass); then
    mv ${prefix}pass ${prefix}fail
  fi
}
run-dir "acl" "openconfig-acl" testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &
run-dir "optical-transport" "openconfig-optical-amplifier" testdata/optical-transport/openconfig-optical-amplifier.yang &
run-dir "optical-transport" "openconfig-transport-line-protection" testdata/optical-transport/openconfig-transport-line-protection.yang &
wait
`,
	}, {
		name:            "basic pyangbind",
		inModelMap:      basicModelMap,
		inValidatorName: "pyangbind",
		wantCmd: `#!/bin/bash
workdir=/workspace/results/pyangbind
mkdir -p "$workdir"
PYANG_MSG_TEMPLATE='messages:{{path:"{file}" line:{line} code:"{code}" type:"{type}" level:{level} message:'"'{msg}'}}"
cmd="$@"
options=(
  -f pybind
  -p testdata
  -p /workspace/third_party/ietf
)
script_options=(
  --msg-template "$PYANG_MSG_TEMPLATE"
)
function run-dir() {
  declare prefix="$workdir"/"$1"=="$2"==
  local output_file="$1"."$2".binding.py
  local cmd_display_options=( --plugindir '$PYANGBIND_PLUGIN_DIR' -o "${output_file}" "${options[@]}" )
  local options=( --plugindir "$PYANGBIND_PLUGIN_DIR" -o "${output_file}" "${options[@]}" )
  shift 2
  echo pyang "${cmd_display_options[@]}" "$@" > ${prefix}cmd
  status=0
  $cmd "${options[@]}" "${script_options[@]}" "$@" &> ${prefix}pass || status=1
  if [[ $status -eq "0" ]]; then
    python "${output_file}" &>> ${prefix}pass || status=1
  fi
  if [[ $status -eq "1" ]]; then
    mv ${prefix}pass ${prefix}fail
  fi
}
run-dir "acl" "openconfig-acl" testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &
run-dir "optical-transport" "openconfig-optical-amplifier" testdata/optical-transport/openconfig-optical-amplifier.yang &
run-dir "optical-transport" "openconfig-transport-line-protection" testdata/optical-transport/openconfig-transport-line-protection.yang &
wait
`,
	}, {
		name:            "basic goyang-ygot",
		inModelMap:      basicModelMap,
		inValidatorName: "goyang-ygot",
		wantCmd: `#!/bin/bash
workdir=/workspace/results/goyang-ygot
mkdir -p "$workdir"
cmd="generator"
options=(
  -path=testdata,/workspace/third_party/ietf
  -package_name=exampleoc -generate_fakeroot -fakeroot_name=device -compress_paths=true
  -shorten_enum_leaf_names -trim_enum_openconfig_prefix -typedef_enum_with_defmod -enum_suffix_for_simple_union_enums
  -exclude_modules=ietf-interfaces -generate_rename -generate_append -generate_getters
  -generate_leaf_getters -generate_delete -annotations -generate_simple_unions
  -list_builder_key_threshold=3
)
script_options=(
)
function run-dir() {
  declare prefix="$workdir"/"$1"=="$2"==
  outdir=$GOPATH/src/ygot/"$1"."$2"/
  mkdir -p "$outdir"
  local options=( -output_file="$outdir"/oc.go "${options[@]}" )
  shift 2
  echo $cmd "${options[@]}" "$@" > ${prefix}cmd
  status=0
  $cmd "${options[@]}" "${script_options[@]}" "$@" &> ${prefix}pass || status=1
  cd "$outdir"
  if [[ $status -eq "0" ]]; then
    go mod init &>> ${prefix}pass || status=1
    go mod tidy &>> ${prefix}pass || status=1
    go build &>> ${prefix}pass || status=1
  fi
  if [[ $status -eq "1" ]]; then
    mv ${prefix}pass ${prefix}fail
  fi
}
run-dir "acl" "openconfig-acl" testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &
run-dir "optical-transport" "openconfig-optical-amplifier" testdata/optical-transport/openconfig-optical-amplifier.yang &
run-dir "optical-transport" "openconfig-transport-line-protection" testdata/optical-transport/openconfig-transport-line-protection.yang &
wait
`,
	}, {
		name:            "basic ygnmi",
		inModelMap:      basicModelMap,
		inValidatorName: "ygnmi",
		wantCmd: `#!/bin/bash
workdir=/workspace/results/ygnmi
mkdir -p "$workdir"
cmd="ygnmi generator"
options=(
  --trim_module_prefix=openconfig
  --exclude_modules=ietf-interfaces
  --split_package_paths="/network-instances/network-instance/protocols/protocol/isis=netinstisis,/network-instances/network-instance/protocols/protocol/bgp=netinstbgp"
  --paths=testdata/...,/workspace/third_party/ietf/...
  --annotations
)
script_options=(
)
function run-dir() {
  declare prefix="$workdir"/"$1"=="$2"==
  outdir=$GOPATH/src/ygnmi/"$1"."$2"
  mkdir -p "$outdir"
  local options=( --output_dir="${outdir}"/oc --base_package_path=ygnmi/"$1"."$2"/oc "${options[@]}" )
  shift 2
  echo $cmd "${options[@]}" "$@" > ${prefix}cmd
  status=0
  $cmd "${options[@]}" "${script_options[@]}" "$@" &> ${prefix}pass || status=1
  if [[ $status -eq "0" ]]; then
    cd "$outdir/oc"
    go mod init &> /dev/null || status=1
    go mod tidy &> /dev/null || status=1
    goimports -w *.go &> /dev/null || status=1
    go build &> /dev/null || status=1
  fi
  if [[ $status -eq "1" ]]; then
    # Only output if there is an error: otherwise the gist comment is too long.
    go build &>> ${prefix}pass || status=1
    mv ${prefix}pass ${prefix}fail
  fi
}
go install golang.org/x/tools/cmd/goimports@latest &>> ${prefix}pass || status=1
run-dir "acl" "openconfig-acl" testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &
run-dir "optical-transport" "openconfig-optical-amplifier" testdata/optical-transport/openconfig-optical-amplifier.yang &
run-dir "optical-transport" "openconfig-transport-line-protection" testdata/optical-transport/openconfig-transport-line-protection.yang &
wait
`,
	}, {
		name:            "basic yanglint",
		inModelMap:      basicModelMap,
		inValidatorName: "yanglint",
		wantCmd: `#!/bin/bash
workdir=/workspace/results/yanglint
mkdir -p "$workdir"
cmd="yanglint"
options=(
  -p testdata
  -p /workspace/third_party/ietf
)
script_options=(
)
function run-dir() {
  declare prefix="$workdir"/"$1"=="$2"==
  shift 2
  echo $cmd "${options[@]}" "$@" > ${prefix}cmd
  if ! $($cmd "${options[@]}" "${script_options[@]}" "$@" &> ${prefix}pass); then
    mv ${prefix}pass ${prefix}fail
  fi
}
run-dir "acl" "openconfig-acl" testdata/acl/openconfig-acl.yang testdata/acl/openconfig-acl-evil-twin.yang &
run-dir "optical-transport" "openconfig-optical-amplifier" testdata/optical-transport/openconfig-optical-amplifier.yang &
run-dir "optical-transport" "openconfig-transport-line-protection" testdata/optical-transport/openconfig-transport-line-protection.yang &
wait
`,
	}, {
		name:            "basic confd",
		inModelMap:      basicModelMap,
		inValidatorName: "confd",
		wantCmd: `#!/bin/bash
workdir=/workspace/results/confd
mkdir -p "$workdir"
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
workdir=/workspace/results/misc-checks
mkdir -p "$workdir"
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
