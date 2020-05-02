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
