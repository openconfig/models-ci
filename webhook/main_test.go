package main

import (
	"os"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestDecodeGitHubJSON(t *testing.T) {
	tests := []struct {
		name    string
		inFile  string
		wantErr bool
		wantOut *githubPullRequestHookInput
	}{{
		name:   "input with valid JSON document",
		inFile: "testdata/valid-openconfig-models.json",
		wantOut: &githubPullRequestHookInput{
			Number: int64(3),
			PullRequest: &githubPullRequest{
				ID:    102817894,
				State: "open",
				Head: &githubPullRequestHead{
					Ref: "newpr2",
					SHA: "509d6be31a46f9577ba069d1be7b61301a310d25",
					Repo: &githubPullRequestRepo{
						FullName: "openconfig/models",
						Name:     "models",
						Owner: &githubRepoOwner{
							Login: "robshakir",
						},
					},
				},
			},
		},
	}, {
		name:    "in with invalid JSON document",
		inFile:  "testdata/invalid-json-file.json",
		wantErr: true,
	}}

	for _, tt := range tests {
		fh, err := os.Open(tt.inFile)
		if err != nil {
			t.Errorf("%s: os.Open(%s): got: %s, want: no error", tt.name, tt.inFile, err)
		}

		got, err := decodeGitHubJSON(fh)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: decodeGitHubJSON(%s): got err: %s, want: no error", tt.name, tt.inFile, err)
				continue
			}
		}

		if diff := pretty.Compare(got, tt.wantOut); diff != "" {
			t.Errorf("%s: decodeGitHubJSON(%s): diff(-got,+want):\n%s", tt.name, tt.inFile, diff)
		}
	}
}

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
			t.Errorf("%s: newGitHubRequestHandler(): did not get valid hash seret, got: %s, want: %s", tt.name, g.hashSecret, tt.wantHashSecret)
		}
	}
}
