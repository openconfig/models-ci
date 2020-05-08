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
	"os"
	"testing"
)

func TestNewGitHubRequestHandler(t *testing.T) {
	tests := []struct {
		name           string
		inEnvSecret    string
		inEnvToken     string
		wantHashSecret string
		wantToken      string
	}{{
		name:           "variables read from environment",
		inEnvSecret:    "testSecret",
		inEnvToken:     "testToken",
		wantHashSecret: "testSecret",
		wantToken:      "testToken",
	}}

	for _, tt := range tests {
		os.Setenv("GITHUB_ACCESS_TOKEN", tt.inEnvToken)
		os.Setenv("GITHUB_SECRET", tt.inEnvSecret)

		g, err := newGitHubRequestHandler()
		if err != nil {
			t.Errorf("%s: newGitHubRequestHandler(): got: %v, want: no error", tt.name, err)
		}

		if g.accessToken != tt.wantToken {
			t.Errorf("%s: newGitHubRequestHandler(): did not get valid access token, got: %s, want: %s", tt.name, g.accessToken, tt.wantToken)
		}

		if g.hashSecret != tt.wantHashSecret {
			t.Errorf("%s: newGitHubRequestHandler(): did not get valid hash secret, got: %s, want: %s", tt.name, g.hashSecret, tt.wantHashSecret)
		}
	}
}
