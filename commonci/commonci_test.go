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

package commonci

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestParseOCModels(t *testing.T) {
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
			got, err := ParseOCModels(tt.inModelRoot)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGetValidatorAndVersionsFromString(t *testing.T) {
	tests := []struct {
		desc       string
		inStr      string
		wantVVList []ValidatorAndVersion
		wantVVMap  map[string]map[string]bool
	}{{
		desc:       "empty",
		inStr:      "",
		wantVVList: nil,
		wantVVMap:  map[string]map[string]bool{},
	}, {
		desc:       "empty with a comma",
		inStr:      ",",
		wantVVList: nil,
		wantVVMap:  map[string]map[string]bool{},
	}, {
		desc:       "single no version",
		inStr:      "pyang",
		wantVVList: []ValidatorAndVersion{{ValidatorId: "pyang"}},
		wantVVMap:  map[string]map[string]bool{"pyang": map[string]bool{"": true}},
	}, {
		desc:       "ending comma",
		inStr:      "pyang,",
		wantVVList: []ValidatorAndVersion{{ValidatorId: "pyang"}},
		wantVVMap:  map[string]map[string]bool{"pyang": map[string]bool{"": true}},
	}, {
		desc:       "ending comma with spaces around before",
		inStr:      "   pyang,   ",
		wantVVList: []ValidatorAndVersion{{ValidatorId: "pyang"}},
		wantVVMap:  map[string]map[string]bool{"pyang": map[string]bool{"": true}},
	}, {
		desc:       "single with version",
		inStr:      "pyang@1.7.2",
		wantVVList: []ValidatorAndVersion{{ValidatorId: "pyang", Version: "1.7.2"}},
		wantVVMap:  map[string]map[string]bool{"pyang": map[string]bool{"1.7.2": true}},
	}, {
		desc:       "single with version and comma",
		inStr:      "pyang@1.7.2,",
		wantVVList: []ValidatorAndVersion{{ValidatorId: "pyang", Version: "1.7.2"}},
		wantVVMap:  map[string]map[string]bool{"pyang": map[string]bool{"1.7.2": true}},
	}, {
		desc:  "more than one version",
		inStr: "pyang@1.7.2,pyang,oc-pyang,pyang@head",
		wantVVList: []ValidatorAndVersion{
			{ValidatorId: "pyang", Version: "1.7.2"},
			{ValidatorId: "pyang", Version: ""},
			{ValidatorId: "oc-pyang", Version: ""},
			{ValidatorId: "pyang", Version: "head"},
		},
		wantVVMap: map[string]map[string]bool{
			"pyang": map[string]bool{
				"":      true,
				"head":  true,
				"1.7.2": true,
			},
			"oc-pyang": map[string]bool{
				"": true,
			},
		},
	}, {
		desc:  "more than one version with ending comma",
		inStr: "pyang@1.7.2,pyang,oc-pyang,pyang@head,",
		wantVVList: []ValidatorAndVersion{
			{ValidatorId: "pyang", Version: "1.7.2"},
			{ValidatorId: "pyang", Version: ""},
			{ValidatorId: "oc-pyang", Version: ""},
			{ValidatorId: "pyang", Version: "head"},
		},
		wantVVMap: map[string]map[string]bool{
			"pyang": map[string]bool{
				"":      true,
				"head":  true,
				"1.7.2": true,
			},
			"oc-pyang": map[string]bool{
				"": true,
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gotVVList, gotVVMap := GetValidatorAndVersionsFromString(tt.inStr)
			if diff := cmp.Diff(gotVVList, tt.wantVVList); diff != "" {
				t.Errorf("[]ValidatorAndVersion (-got, +want):\n%s", diff)
			}
			if diff := cmp.Diff(gotVVMap, tt.wantVVMap); diff != "" {
				t.Errorf("ValidatorAndVersion Map (-got, +want):\n%s", diff)
			}
		})
	}
}

func TestValidatorAndVersionsDiff(t *testing.T) {
	tests := []struct {
		desc    string
		inAStr  string
		inBStr  string
		wantStr string
	}{{
		desc:    "single no version",
		inAStr:  "pyang",
		inBStr:  "pyang",
		wantStr: "",
	}, {
		desc:    "single with ending comma",
		inAStr:  "pyang,",
		inBStr:  "pyang",
		wantStr: "",
	}, {
		desc:    "single version doesn't match latest",
		inAStr:  "pyang@1.7.2",
		inBStr:  "pyang",
		wantStr: "pyang@1.7.2",
	}, {
		desc:    "single version doesn't match",
		inAStr:  "pyang@1.7.2",
		inBStr:  "pyang@1.7.1",
		wantStr: "pyang@1.7.2",
	}, {
		desc:    "single with version",
		inAStr:  "pyang@1.7.2",
		inBStr:  "pyang@1.7.2",
		wantStr: "",
	}, {
		desc:    "more than one version",
		inAStr:  "pyang@1.7.2,pyang,oc-pyang,pyang@head",
		inBStr:  "pyang@head",
		wantStr: "pyang@1.7.2,pyang,oc-pyang",
	}, {
		desc:    "more than one version deletes",
		inAStr:  "pyang@1.7.2,pyang,oc-pyang,pyang@head",
		inBStr:  "oc-pyang,pyang@1.7.2",
		wantStr: "pyang,pyang@head",
	}, {
		desc:    "more than one version deletes with deleting something not there",
		inAStr:  "pyang@1.7.2,pyang,oc-pyang,pyang@head",
		inBStr:  "oc-pyang,pyang@1.7.2,super-check",
		wantStr: "pyang,pyang@head",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if gotStr := ValidatorAndVersionsDiff(tt.inAStr, tt.inBStr); gotStr != tt.wantStr {
				t.Errorf("got %q, want %q", gotStr, tt.wantStr)
			}
		})
	}
}
