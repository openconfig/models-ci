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
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// GithubRequestHandler carries information relating to the GitHub session that
// is being used for the continuous integration.
type GithubRequestHandler struct {
	// hashSecret is the GitHub secret that is specified with the hook, it is
	// used to validate whether the response that is received is from GitHub.
	hashSecret string
	// Client is the connection to GitHub that should be utilised.
	client *github.Client
	// accessToken is the OAuth token that should be used for interactions with
	// the GitHub API and to retrieve repo contents.
	accessToken string
	labels      map[string]bool
}

// GithubPRUpdate is used to specify how an update to the status of a PR should
// be made with the UpdatePRStatus method.
type GithubPRUpdate struct {
	Owner       string
	Repo        string
	Ref         string
	NewStatus   string
	URL         string
	Description string
	Context     string
}

// Retry retries a function maxN times or when it returns true.
// In between each retry there is a small delay.
// This is intended to be used for posting results to GitHub from GCB, which
// frequently experiences errors likely due to connection issues.
func Retry(maxN uint, name string, f func() error) error {
	var err error
	for i := uint(0); i <= maxN; i++ {
		if err = f(); err == nil {
			return nil
		}
		log.Printf("Retry %d of %s, error: %v", i, name, err)
		time.Sleep(250 * time.Millisecond)
	}
	return err
}

// CreateCIOutputGist creates a GitHub Gist, and appends a comment with the
// result of the validator into it.  The function returns the URL and ID of the
// Gist that was created, and an error if experienced during processing.
func (g *GithubRequestHandler) CreateCIOutputGist(description, content string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	public := false
	// Create a new Gist struct - the description is used as the tag-line of
	// the created content, and the GistFilename (within the Files map) as
	// the input filename in the GitHub UI.
	gist := &github.Gist{
		Description: &description,
		Public:      &public,
		Files: map[github.GistFilename]github.GistFile{
			"oc-ci-run": {Content: &content},
		},
	}

	if err := Retry(5, fmt.Sprintf("gist creation for %s with content\n%s\n", description, content), func() error {
		var err error
		gist, _, err = g.client.Gists.Create(ctx, gist)
		return err
	}); err != nil {
		return "", "", fmt.Errorf("could not create gist: %s", err)
	}
	return *gist.HTMLURL, *gist.ID, nil
}

// AddGistComment adds a comment to a gist and returns its ID.
func (g *GithubRequestHandler) AddGistComment(gistID, title, output string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	gistComment := fmt.Sprintf("# %s\n%s", title, output)

	var id int64
	if err := Retry(5, "gist comment creation", func() error {
		c, _, err := g.client.Gists.CreateComment(ctx, gistID, &github.GistComment{Body: &gistComment})
		if err != nil {
			return err
		}
		id = c.GetID()
		return nil
	}); err != nil {
		return "", err
	}
	return id, nil
}

// UpdatePRStatus takes an input githubPRUpdate struct and updates a GitHub
// pull request's status with the relevant details. It returns an error if
// the update was not successful.
func (g *GithubRequestHandler) UpdatePRStatus(update *GithubPRUpdate) error {
	if !validStatuses[update.NewStatus] {
		return fmt.Errorf("invalid status %s", update.NewStatus)
	}

	if update.NewStatus == "" || update.Repo == "" || update.Ref == "" || update.Owner == "" {
		return fmt.Errorf("must specify required fields (status (%s), repo (%s), reference (%s) and owner (%s)) for update", update.NewStatus, update.Repo, update.Ref, update.Owner)
	}

	// The go-github library takes string pointers within the struct, and hence
	// we have to provide everything as a pointer.
	status := &github.RepoStatus{
		State:       &update.NewStatus,
		TargetURL:   &update.URL,
		Description: &update.Description,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	// Context is an optional argument.
	if update.Context != "" {
		status.Context = &update.Context
	}

	if update.Description != "" {
		status.Description = &update.Description
	}

	return Retry(5, "PR status update", func() error {
		_, _, err := g.client.Repositories.CreateStatus(ctx, update.Owner, update.Repo, update.Ref, status)
		return err
	})
}

// IsPRApproved checks whether a PR is approved or not.
// TODO(wenbli): If the SkipIfNotApproved feature is used, this function should
// undergo testing due to having some logic.
// unit tests can be created based onon actual models-ci repo data that's sent back for a particular PR.
func (g *GithubRequestHandler) IsPRApproved(owner, repo string, prNumber int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel() // cancel context if the function returns before the timeout
	var reviews []*github.PullRequestReview
	if err := Retry(5, "get PR reviews list", func() error {
		var err error
		reviews, _, err = g.client.PullRequests.ListReviews(ctx, owner, repo, prNumber, nil)
		return err
	}); err != nil {
		return false, err
	}

	for i := len(reviews) - 1; i != -1; i-- {
		review := reviews[i]
		switch strings.ToLower(review.GetState()) {
		case "approved":
			return true, nil
		case "changes_requested":
			return false, nil
		}
	}
	return false, nil
}

// PostLabel posts the given label to the PR. It is idempotent.
// unit tests can be created based onon actual models-ci repo data that's sent back.
func (g *GithubRequestHandler) PostLabel(labelName, labelColor, owner, repo string, prNumber int) error {
	if g.labels[labelName] {
		// Label already exists.
		return nil
	}

	label := &github.Label{Name: &labelName, Color: &labelColor}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Label may very well already exist within the repo, so skip creation if we see it.
	_, _, err := g.client.Issues.GetLabel(ctx, owner, repo, labelName)
	if err != nil {
		if err := Retry(5, "creating label", func() error {
			_, _, err = g.client.Issues.CreateLabel(ctx, owner, repo, label)
			return err
		}); err != nil {
			return err
		}
	}

	err = Retry(5, "adding label to PR", func() error {
		_, _, err = g.client.Issues.AddLabelsToIssue(ctx, owner, repo, prNumber, []string{labelName})
		return err
	})
	if err == nil {
		g.labels[labelName] = true
	}

	return err
}

// DeleteLabel removes the given label from the PR. It does not remove the
// label from the repo.
func (g *GithubRequestHandler) DeleteLabel(labelName, owner, repo string, prNumber int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	if err := Retry(5, "removing label from PR", func() error {
		_, err := g.client.Issues.RemoveLabelForIssue(ctx, owner, repo, prNumber, labelName)
		return err
	}); err != nil {
		return err
	}

	// Do not take the second step to delete the label from the repo as
	// we're only interested in deleting the label from the PR.

	delete(g.labels, labelName)
	return nil
}

// AddPRComment posts a comment to the PR.
func (g *GithubRequestHandler) AddPRComment(body *string, owner, repo string, prNumber int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	if err := Retry(5, "posting issue comment to PR", func() error {
		_, _, err := g.client.Issues.CreateComment(ctx, owner, repo, prNumber, &github.IssueComment{Body: body})
		return err
	}); err != nil {
		return err
	}
	return nil
}

// NewGitHubRequestHandler sets up a new GithubRequestHandler struct which
// creates an oauth2 client with a GitHub access token (as specified by the
// GITHUB_ACCESS_TOKEN environment variable), and a connection to the GitHub
// API through the github.com/google/go-github/github library. It returns the
// initialised GithubRequestHandler struct, or an error as to why the
// initialisation failed.
func NewGitHubRequestHandler() (*GithubRequestHandler, error) {
	accesstk := os.Getenv("GITHUB_ACCESS_TOKEN")
	if accesstk == "" {
		return nil, errors.New("newGitHubRequestHandler: invalid access token environment variable set")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accesstk},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	// Set the timeout for the oauth client such that we do not hang around
	// waiting for the client to complete.
	tc.Timeout = 2 * time.Second

	// Create a new GitHub client using the go-github library.
	client := github.NewClient(tc)
	return &GithubRequestHandler{
		// If the environment variable GITHUB_SECRET was set then we store it in
		// the struct, this is a secret that is used to calculate a hash of the
		// message so that we can validate it.
		client:      client,
		accessToken: accesstk,
		labels:      map[string]bool{},
	}, nil
}
