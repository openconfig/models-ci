package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/models-ci/commonci"
)

func TestGenerateScript(t *testing.T) {
	tests := []struct {
		desc                             string
		inMap                            commonci.OpenConfigModelMap
		inOutDir, inRepoDir, inPluginDir string
		wantScript                       string
		wantIndex                        indexMap
		wantErr                          bool
	}{{
		desc: "single entry",
		inMap: commonci.OpenConfigModelMap{
			ModelInfoMap: map[string][]commonci.ModelInfo{
				"model": {{
					Name:     "openconfig-model",
					DocFiles: []string{"a.yang", "b.yang"},
				}},
			},
		},
		inOutDir:    "out",
		inRepoDir:   "repo",
		inPluginDir: "plugins",
		wantScript: `#!/bin/bash
pyang --plugindir=plugins/openconfig_pyang/plugins/ \
  -p repo \
  --doc-format=html \
  -o out/openconfig-model.html \
  -f docs \
  repo/a.yang \
  repo/b.yang
pyang --plugindir=plugins/openconfig_pyang/plugins/ \
  -p repo \
  -o out/openconfig-model-tree.html \
  -f oc-jstree \
  --oc-jstree-strip \
  repo/a.yang \
  repo/b.yang`,
		wantIndex: indexMap{
			"openconfig-model": map[string]string{
				"docs": "out/openconfig-model.html",
				"tree": "out/openconfig-model-tree.html",
			},
		},
	}, {
		desc: "rule with no docs",
		inMap: commonci.OpenConfigModelMap{
			ModelInfoMap: map[string][]commonci.ModelInfo{
				"model": {{
					Name:       "openconfig-model",
					BuildFiles: []string{"a.yang", "b.yang"},
				}},
			},
		},
		inOutDir:    "out",
		inRepoDir:   "repo",
		inPluginDir: "plugins",
		wantScript:  `#!/bin/bash`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gotScript, gotIndex, err := generateScript(tt.inMap, tt.inOutDir, tt.inRepoDir, tt.inPluginDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("did not get expected error, got: %v, wantErr? %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(gotScript.String(), tt.wantScript); diff != "" {
				t.Fatalf("did not get expected script, difF(-got,+want):\n%s", diff)
			}
			if diff := cmp.Diff(gotIndex, tt.wantIndex, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("did not get expected index, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
