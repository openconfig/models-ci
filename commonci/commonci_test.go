package commonci

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
		name:        "variables read from environment",
		inEnvSecret: "testSecret",
		inEnvToken:  "testToken",
		wantToken:   "testToken",
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
	}
}
