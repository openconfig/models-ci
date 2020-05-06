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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/github"
	"github.com/openconfig/gnmi/errdiff"
)

// NOTE: fake HTTP server objects are copied from go-github repo because they're unexported.
const (
	// baseURLPath is a non-empty Client.BaseURL path to use during tests,
	// to ensure relative URLs are used for all endpoints. See issue #752.
	baseURLPath = "/api-v3"
)

func TestRetry(t *testing.T) {
	var tryNum int
	tests := []struct {
		name         string
		inExtraTries uint
		inFunc       func() error
		// wantErrSubstr is empty means no error expected.
		wantErrSubstr string
	}{{
		name:         "pass on first try",
		inExtraTries: 2,
		inFunc:       func() error { return nil },
	}, {
		name:         "pass on second try",
		inExtraTries: 2,
		inFunc: func() error {
			tryNum++
			if tryNum == 2 {
				return nil
			}
			return fmt.Errorf("error msg")
		},
	}, {
		name:         "pass on third try",
		inExtraTries: 2,
		inFunc: func() error {
			tryNum++
			if tryNum == 3 {
				return nil
			}
			return fmt.Errorf("error msg")
		},
	}, {
		name:         "fail after third try",
		inExtraTries: 2,
		inFunc: func() error {
			tryNum++
			if tryNum == 4 {
				return nil
			}
			return fmt.Errorf("error msg")
		},
		wantErrSubstr: "error msg",
	}}

	for _, tt := range tests {
		tryNum = 0
		t.Run(tt.name, func(t *testing.T) {
			err := Retry(tt.inExtraTries, tt.name, tt.inFunc)
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Errorf("did not get expected error, %s", diff)
			}
		})
	}
}

// setup sets up a test HTTP server along with a github.Client that is
// configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup() (client *github.Client, mux *http.ServeMux, serverURL string, teardown func()) {
	// mux is the HTTP request multiplexer used with the test server.
	mux = http.NewServeMux()

	// We want to ensure that tests catch mistakes where the endpoint URL is
	// specified as absolute rather than relative. It only makes a difference
	// when there's a non-empty base URL path. So, use that. See issue #752.
	apiHandler := http.NewServeMux()
	apiHandler.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		fmt.Fprintln(os.Stderr, "\tSee https://github.com/google/go-github/issues/752 for information.")
		http.Error(w, "Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the GitHub client being tested and is
	// configured to use test server.
	client = github.NewClient(nil)
	url, _ := url.Parse(server.URL + baseURLPath + "/")
	client.BaseURL = url
	client.UploadURL = url

	return client, mux, server.URL, server.Close
}

func testMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func TestPostLabel(t *testing.T) {
	var labelRetrieved, labelCreated, labelAdded bool
	// parameters used across queries.
	labelName := "n"
	labelColor := "c"
	tests := []struct {
		name                string
		inKnownLabels       []string
		inLabelExistsInRepo bool
		inLabelCreateFails  bool
		inLabelAddFails     bool
		wantLabelRetrieved  bool
		wantLabelCreated    bool
		wantLabelAdded      bool
		wantErr             bool
	}{{
		name:                "label not already seen, exists in repo",
		inKnownLabels:       nil,
		inLabelExistsInRepo: true,
		wantLabelRetrieved:  true,
		wantLabelCreated:    false,
		wantLabelAdded:      true,
	}, {
		name:               "label already seen",
		inKnownLabels:      []string{"n", "a"},
		wantLabelRetrieved: false,
		wantLabelCreated:   false,
		wantLabelAdded:     false,
	}, {
		name:                "label not already seen, doesn't exist in repo",
		inKnownLabels:       nil,
		inLabelExistsInRepo: false,
		wantLabelRetrieved:  false,
		wantLabelCreated:    true,
		wantLabelAdded:      true,
	}, {
		name:                "label add fails",
		inKnownLabels:       nil,
		inLabelExistsInRepo: false,
		inLabelAddFails:     true,
		wantLabelRetrieved:  false,
		wantLabelCreated:    true,
		wantLabelAdded:      false,
		wantErr:             true,
	}, {
		name:                "label create fails",
		inKnownLabels:       nil,
		inLabelExistsInRepo: false,
		inLabelCreateFails:  true,
		wantLabelRetrieved:  false,
		wantLabelCreated:    false,
		wantLabelAdded:      false,
		wantErr:             true,
	}}

	for _, tt := range tests {
		labelCreated = false
		labelAdded = false
		labelRetrieved = false

		t.Run(tt.name, func(t *testing.T) {
			client, mux, _, teardown := setup()
			defer teardown()

			g := &GithubRequestHandler{
				client: client,
				labels: map[string]bool{},
			}
			for _, label := range tt.inKnownLabels {
				g.labels[label] = true
			}

			// GetLabel response
			if tt.inLabelExistsInRepo {
				mux.HandleFunc("/repos/o/r/labels/n", func(w http.ResponseWriter, r *http.Request) {
					testMethod(t, r, "GET")
					fmt.Fprint(w, `{"url":"u", "name": "n", "color": "c", "description": "d"}`)

					labelRetrieved = true
				})
			}

			// CreateLabel response
			if !tt.inLabelCreateFails {
				mux.HandleFunc("/repos/o/r/labels", func(w http.ResponseWriter, r *http.Request) {
					v := new(github.Label)
					json.NewDecoder(r.Body).Decode(v)

					testMethod(t, r, "POST")
					if want := (&github.Label{Name: &labelName, Color: &labelColor}); !cmp.Equal(v, want) {
						t.Errorf("Request body = %+v, want %+v", v, want)
					}

					labelCreated = true
				})
			}

			// AddLabelsToIssue response
			if !tt.inLabelAddFails {
				mux.HandleFunc("/repos/o/r/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
					var v []string
					json.NewDecoder(r.Body).Decode(&v)

					testMethod(t, r, "POST")
					if want := []string{labelName}; !cmp.Equal(v, want) {
						t.Errorf("Request body = %+v, want %+v", v, want)
					}

					labelAdded = true
				})
			}

			err := g.PostLabel(labelName, labelColor, "o", "r", 1)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Got err: %v, wantErr: %v", err, tt.wantErr)
			}
			if labelRetrieved != tt.wantLabelRetrieved {
				t.Errorf("GetLabel happened?, got %v, want %v", labelRetrieved, tt.wantLabelRetrieved)
			}
			if labelCreated != tt.wantLabelCreated {
				t.Errorf("CreateLabel happened?, got %v, want %v", labelCreated, tt.wantLabelCreated)
			}
			if labelAdded != tt.wantLabelAdded {
				t.Errorf("AddLabelsToIssue happened?, got %v, want %v", labelAdded, tt.wantLabelAdded)
			}

			// Check label added to known set of labels.
			if labelAdded {
				wantAllLabels := map[string]bool{}
				for _, n := range tt.inKnownLabels {
					wantAllLabels[n] = true
				}
				wantAllLabels["n"] = true
				if diff := cmp.Diff(wantAllLabels, g.labels); diff != "" {
					t.Errorf("known labels map not updated (-want, +got):\n%s", diff)
				}
			}
		})
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
		name:        "variables read from environment",
		inEnvSecret: "testSecret",
		inEnvToken:  "testToken",
		wantToken:   "testToken",
	}}

	for _, tt := range tests {
		os.Setenv("GITHUB_ACCESS_TOKEN", tt.inEnvToken)
		os.Setenv("GITHUB_SECRET", tt.inEnvSecret)

		g, err := NewGitHubRequestHandler()
		if err != nil {
			t.Errorf("%s: newGitHubRequestHandler(): got: %v, want: no error", tt.name, err)
		}

		if g.accessToken != tt.wantToken {
			t.Errorf("%s: newGitHubRequestHandler(): did not get valid access token, got: %s, want: %s", tt.name, g.accessToken, tt.wantToken)
		}
	}
}
