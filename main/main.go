package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/oauth2"

	glog "github.com/golang/glog"
	"github.com/google/go-github/github"
)

var (
	// validStatuses are the status codes that are valid in the GitHub UI for a
	// pull request status.
	validStatuses = map[string]bool{
		"pending": true,
		"success": true,
		"error":   true,
		"failure": true,
	}

	// ciName is the name of the CI as it appears on the GitHub PR page.
	ciName = "OpenConfig CI Experimental"

	repoSlug = flag.String("repo-slug", "openconfig/public", "repo where CI is run")

	// repoInfo contains the description of the expected repo to be tested
	// under CI. This information is parsed from the input flag, and is
	// verified against the Travis environment variable TRAVIS_PULL_REQUEST_SLUG
	repoInfo RepoInfo

	// lintOutputPath stores the path at which the output of the linter is stored.
	// By default this should be /tmp/lint.out.
	lintOutputPath = flag.String("lint-output-path", "/tmp/lint.out", "path where the linter script output is stored")
)

type RepoInfo struct {
	// repoSlug is the full name of the GitHub repository in the form "owner/repo"
	repoSlug string
	// owner is the owner of the repository as parsed from the repoSlug.
	owner string
	// repo is the owner of the repository as parsed from the repoSlug.
	repo string
}

func newRepoInfo(repoSlug string) RepoInfo {
	repoSplit := strings.Split(repoSlug, "/")
	return RepoInfo{
		repoSlug: repoSlug,
		owner:    repoSplit[0],
		repo:     repoSplit[1],
	}
}

// githubCIHandler carries information relating to the GitHub session that
// is being used for the continuous integration.
type githubCIHandler struct {
	// client is the connection to GitHub that should be utilised.
	client *github.Client
	// accessToken is the OAuth token that should be used for interactions with
	// the GitHub API and to retrieve repo contents.
	accessToken string
	// lintTestPath is the path to where the output of the OC linter can be found.
	lintTestPath string
}

// githubPRUpdate is used to specify how an update to the status of a PR should
// be made with the updatePRStatus method.
type githubPRUpdate struct {
	Owner       string
	Repo        string
	Ref         string
	NewStatus   string
	URL         string
	Description string
	Context     string
}

// runLintTests runs lint tests for a pull request on Travis-CI.
// list of current (lint test / result location):
// - openconfig_pyang extensions / PR gist.
func (g *githubCIHandler) runLintTests() {
	travisRepoSlug := os.Getenv("TRAVIS_PULL_REQUEST_SLUG")
	if travisRepoSlug == "" {
		// TODO(wenbli): We should continue to lint without posting to a PR for pushes in case someone pushes.
		glog.Info("Skip linting for push request")
		return
	} else if travisRepoSlug != repoInfo.repoSlug {
		// Ensure that we're running in the expected repo as requested by the build script.
		glog.Errorf("Not processing pull request for %q as it is not the expected input repo %q", travisRepoSlug, repoInfo.repoSlug)
		os.Exit(1)
		return
	}

	commitSHA := os.Getenv("TRAVIS_PULL_REQUEST_SHA")
	branch := os.Getenv("TRAVIS_PULL_REQUEST_BRANCH")
	// Update the status to pending so that the user can see that we have received
	// this request and are ready to run the CI.
	glog.Infof("run CI for commit %s, branch %s", commitSHA, branch)
	update := &githubPRUpdate{
		Owner:       repoInfo.owner,
		Repo:        repoInfo.repo,
		Ref:         commitSHA,
		Description: ciName + " Running",
		NewStatus:   "pending",
		Context:     ciName,
	}
	if err := g.updatePRStatus(update); err != nil {
		glog.Errorf("couldn't update PR: %s", err)
	}

	// Launch a go routine to run the PR CI.
	g.runLintGoTests(repoInfo, commitSHA)
}

// runLintGoTests runs the OpenConfig linter, and Go-based tests for the models
// repo. The results are written to a GitHub Gist, and into the PR that was
// modified, associated with the commit reference SHA.
func (g *githubCIHandler) runLintGoTests(repoInfo RepoInfo, sha string) {

	// Run the tests using exec.
	lintCmd := exec.Command("make", "clean", "lint_html")

	out, ciErr := lintCmd.CombinedOutput()
	glog.Infof("Lint test output: %s", out)

	lintOK := true
	if ciErr != nil {
		lintOK = false
	}

	output := fmt.Sprintf("%s", string(out))
	url, _, err := g.createCIOutputGist(output, lintOK)
	if err != nil {
		glog.Errorf("couldn't create gist: %s", err)
	}

	prUpdate := &githubPRUpdate{
		Owner:   repoInfo.owner,
		Repo:    repoInfo.repo,
		Ref:     sha,
		URL:     url,
		Context: ciName,
	}

	if ciErr != nil {
		prUpdate.NewStatus = "failure"
		prUpdate.Description = ciName + " Failed"

		if uperr := g.updatePRStatus(prUpdate); uperr != nil {
			glog.Errorf("couldn't update PR to failed: %s\nerror: %s", out, uperr)
		}
		return
	}

	prUpdate.NewStatus = "success"
	prUpdate.Description = ciName + " Succeeded"
	if uperr := g.updatePRStatus(prUpdate); uperr != nil {
		glog.Errorf("couldn't update PR to succeeded: %s", uperr)
	}
}

// createCIOutputGist creates a GitHub Gist, and appends comment output to it.
// In this case, the PR number is used as the title for the Gist (to identify the
// changes), output is the stdout/stderr output of the CI test, and success
// indicates whether it was a successful test. The output of the /tmp/lint.out
// file is taken and this is posted as a Gist comment. The function
// returns the URL and ID of the Gist that was created or the error
// experienced during processing.
func (g *githubCIHandler) createCIOutputGist(output string, lintOK bool) (string, string, error) {

	d := fmt.Sprintf(ciName+" Test Run Output for PR %v", os.Getenv("TRAVIS_PULL_REQUEST"))
	f := false

	// Create a new Gist struct - the description is used as the tag-line of
	// the created content, and the GistFilename (within the Files map) as
	// the input filename in the GitHub UI.
	gist := &github.Gist{
		Description: &d,
		Public:      &f,
		Files: map[github.GistFilename]github.GistFile{
			"oc-ci-run": {Content: &output},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	gisto, _, err := g.client.Gists.Create(ctx, gist)
	if err != nil {
		return "", "", fmt.Errorf("could not create gist: %s", err)
	}

	// Read the output of the linter for the CI from a file created on disk.
	// TODO(robjs): should this be a dynamic filename so that we cannot inject
	// any content into it? The exposure is low since this will just be written
	// to GitHub.
	lintOut, err := ioutil.ReadFile(g.lintTestPath)
	if err != nil {
		return "", "", err
	}

	// The title of the comment uses the relevant emoji to show whether it
	// succeeded or failed - so populate this based on the success of the test.
	lintSymbol := ":white_check_mark:"
	if !lintOK {
		lintSymbol = ":no_entry:"
	}
	s := fmt.Sprintf("# %s Lint\n%s", lintSymbol, string(lintOut))
	if _, _, err = g.client.Gists.CreateComment(ctx, *gisto.ID, &github.GistComment{Body: &s}); err != nil {
		return "", "", err
	}

	return *gisto.HTMLURL, *gisto.ID, nil
}

// updatePRStatus takes an input githubPRUpdate struct and updates a GitHub
// pull request's status with the relevant details. It returns an error if
// the update was not successful.
func (g *githubCIHandler) updatePRStatus(update *githubPRUpdate) error {
	if !validStatuses[update.NewStatus] {
		return fmt.Errorf("invalid status %s", update.NewStatus)
	}

	if update.NewStatus == "" || update.Repo == "" || update.Ref == "" || update.Owner == "" {
		return fmt.Errorf("must specify required fields (status (%s), repo (%s), reference (%s) and owner (%s)) for update",
			update.NewStatus, update.Repo, update.Ref, update.Owner)
	}

	// The go-github library takes string pointers within the struct, and hence
	// we have to provide everything as a pointer.
	status := &github.RepoStatus{
		State:       &update.NewStatus,
		TargetURL:   &update.URL,
		Description: &update.Description,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	// Context is an optional argument.
	if update.Context != "" {
		status.Context = &update.Context
	}

	if update.Description != "" {
		status.Description = &update.Description
	}

	if _, _, err := g.client.Repositories.CreateStatus(ctx, update.Owner, update.Repo, update.Ref, status); err != nil {
		return err
	}
	return nil
}

// newGitHubCIHandler sets up a new githubCIHandler struct which
// creates an oauth2 client with a GitHub access token (as specified by the
// GITHUB_ACCESS_TOKEN environment variable), and a connection to the GitHub
// API through the github.com/google/go-github/github library. It returns the
// initialised githubCIHandler struct, or an error as to why the
// initialisation failed.
func newGitHubCIHandler() (*githubCIHandler, error) {
	accesstk := os.Getenv("GITHUB_ACCESS_TOKEN")
	if accesstk == "" {
		return nil, errors.New("invalid access token environment variable set")
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
	return &githubCIHandler{
		client:       client,
		accessToken:  accesstk,
		lintTestPath: *lintOutputPath,
	}, nil
}

func main() {
	flag.Parse()
	repoInfo = newRepoInfo(*repoSlug)

	h, err := newGitHubCIHandler()
	if err != nil {
		glog.Errorf("Could not initialise GitHub client: %v", err)
		return
	}

	h.runLintTests()
}
