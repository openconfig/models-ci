package main

import (
	"os"
	"testing"
)

func TestNewGitHubRequestHandler(t *testing.T) {
	tests := []struct {
		name       string
		inEnvToken string
		wantToken  string
	}{{
		name:       "variables read from environment",
		inEnvToken: "testToken",
		wantToken:  "testToken",
	}}

	for _, tt := range tests {
		os.Setenv("GITHUB_ACCESS_TOKEN", tt.inEnvToken)

		g, err := newGitHubCIHandler()
		if err != nil {
			t.Errorf("%s: newGitHubCIHandler(): got: %v, want: no error", tt.name, err)
		}

		if g.accessToken != tt.wantToken {
			t.Errorf("%s: newGitHubCIHandler(): did not get valid access token, got: %s, want: %s", tt.name, g.accessToken, tt.wantToken)
		}
	}
}
