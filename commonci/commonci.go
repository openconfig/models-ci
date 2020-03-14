package commonci

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// commonci contains definitions and constants common to the CI process in
// general (esp. cmd_gen and post_result scripts).

const (
	RootDir        = "/workspace"
	ResultsDir     = "/workspace/results"
	ScriptFileName = "script.sh"
	FailFileName   = "fail"
	OutFileName    = "out"
)

// ValidatorResultsDir determines where a particular validator's results are
// stored.
func ValidatorResultsDir(validatorId, version string) string {
	return filepath.Join(ResultsDir, validatorId+version)
}

// Validator describes a validation tool.
type Validator struct {
	// The longer name of the validator.
	Name string
	// IsPerModel means the validator is run per-model, not across the
	// entire repo of YANG files.
	IsPerModel bool
	// RunBeforeApproval means to run the test on a PR even before approval
	// status. Longer tests are best be omitted from this category.
	RunBeforeApproval bool
}

var (
	// Validators contains the set of supported validators to be run under CI.
	// The key is a unique identifier that's safe to use as a directory name.
	Validators = map[string]*Validator{
		"pyang": &Validator{
			Name:              "Pyang",
			IsPerModel:        true,
			RunBeforeApproval: true,
		},
		"oc-pyang": &Validator{
			Name:              "OpenConfig Linter",
			IsPerModel:        true,
			RunBeforeApproval: true,
		},
		"pyangbind": &Validator{
			Name:              "Pyangbind",
			IsPerModel:        true,
			RunBeforeApproval: true,
		},
		"goyang-ygot": &Validator{
			Name:       "goyang/ygot",
			IsPerModel: true,
			// This is ideally false, but GCB can't rebuild GitHub App builds more than 3
			// days ago, so the current way of asking users to rebuild doesn't work.
			RunBeforeApproval: true,
		},
		"yanglint": &Validator{
			Name:              "yanglint",
			IsPerModel:        true,
			RunBeforeApproval: true,
		},
		"regexp": &Validator{
			Name:              "regexp tests",
			IsPerModel:        false,
			RunBeforeApproval: true,
		},
	}

	// LabelColors are some helper hex colours for posting to GitHub.
	LabelColors = map[string]string{
		"yellow": "ffe200",
		"red":    "ff0000",
		"orange": "ffa500",
		"blue":   "00bfff",
	}

	// validStatuses are the valid pull request status codes that are valid in the GitHub UI.
	validStatuses = map[string]bool{
		"pending": true,
		"success": true,
		"error":   true,
		"failure": true,
	}
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
func Retry(maxN uint, name string, f func() error) {
	for i := uint(0); i != maxN; i++ {
		err := f()
		if err == nil {
			return
		}
		log.Printf("Retry %d of %s, error: %v", i, name, err)
		time.Sleep(250 * time.Millisecond)
	}
}

// CreateCIOutputGist creates a GitHub Gist, and appends comment output to it.
// In this case, the runID is used as the title for the Gist (to identify the
// changes), output is the stdout/stderr output of the CI test, and success
// indicates whether it was a successful test. The output of the /tmp/lint.out
// file is taken and this is posted as a Gist comment, along with the contents
// of the /tmp/go-tests.out file which contains other unit tests. The function
// returns the URL and ID of the Gist that was created, its ID or the error
// experienced during processing.
func (g *GithubRequestHandler) CreateCIOutputGist(validatorId, version string) (string, string, error) {
	d := fmt.Sprintf(Validators[validatorId].Name + version + " Test Run Script")
	public := false

	outBytes, err := ioutil.ReadFile(filepath.Join(ValidatorResultsDir(validatorId, version), OutFileName))
	if err != nil {
		return "", "", err
	}
	outString := string(outBytes)
	if outString == "" {
		outString = "No output"
	}

	// Create a new Gist struct - the description is used as the tag-line of
	// the created content, and the GistFilename (within the Files map) as
	// the input filename in the GitHub UI.
	gist := &github.Gist{
		Description: &d,
		Public:      &public,
		Files: map[github.GistFilename]github.GistFile{
			"oc-ci-run": {Content: &outString},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	Retry(5, fmt.Sprintf("gist creation for %s with content\n%s\n", validatorId+version, outString), func() error {
		gist, _, err = g.client.Gists.Create(ctx, gist)
		return err
	})
	if err != nil {
		return "", "", fmt.Errorf("could not create gist: %s", err)
	}
	return *gist.HTMLURL, *gist.ID, nil
}

// AddGistComment adds a comment to a gist.
func (g *GithubRequestHandler) AddGistComment(gistID string, output string, title string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	s := fmt.Sprintf("# %s\n%s", title, output)

	var err error
	Retry(5, "gist comment creation", func() error {
		_, _, err = g.client.Gists.CreateComment(ctx, gistID, &github.GistComment{Body: &s})
		return err
	})
	return err

	// XXX: Unfortunately check runs are currently unsupported by GCB.
	//      Check runs is a better UI than posting gists as statuses.
	//      https://groups.google.com/g/google-cloud-dev/c/fON-kDlykLc
	// status := "completed"
	// conclusion := "neutral"
	// summary := "this is a test of the check run creation API"
	// checkRunOpts := github.CreateCheckRunOptions{
	// 	Name:       title,
	// 	HeadSHA:    commitSHA,
	// 	Status:     &status,
	// 	Conclusion: &conclusion,
	// 	Output: &github.CheckRunOutput{
	// 		Title:   &title,
	// 		Summary: &summary,
	// 		Text:    &output,
	// 	},
	// }

	// ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel2() // cancel context if the function returns before the timeout

	// checkRun, resp, err := g.client.Checks.CreateCheckRun(ctx2, owner, repo, checkRunOpts)
	// log.Print(resp)
	// log.Print(*resp.Response)
	// log.Print(checkRun)
	// return err
}

// UpdatePRStatus takes an input githubPRUpdate struct and updates a GitHub
// pull request's status with the relevant details. It returns an error if
// the update was not successful.
func (g *GithubRequestHandler) UpdatePRStatus(update *GithubPRUpdate) error {
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
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	// Context is an optional argument.
	if update.Context != "" {
		status.Context = &update.Context
	}

	if update.Description != "" {
		status.Description = &update.Description
	}

	var err error
	Retry(5, "PR status update", func() error {
		_, _, err := g.client.Repositories.CreateStatus(ctx, update.Owner, update.Repo, update.Ref, status)
		return err
	})
	return err
}

// IsPRApproved checks whether a PR is approved or not.
func (g *GithubRequestHandler) IsPRApproved(owner, repo string, prNumber int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel() // cancel context if the function returns before the timeout
	var err error
	var reviews []*github.PullRequestReview
	Retry(5, "get PR reviews list", func() error {
		reviews, _, err = g.client.PullRequests.ListReviews(ctx, owner, repo, prNumber, nil)
		return err
	})
	if err != nil {
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
		Retry(5, "creating label", func() error {
			_, _, err = g.client.Issues.CreateLabel(ctx, owner, repo, label)
			return err
		})
		if err != nil {
			return err
		}
	}

	Retry(5, "adding label to PR", func() error {
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
	var err error
	Retry(5, "removing label from PR", func() error {
		_, err = g.client.Issues.RemoveLabelForIssue(ctx, owner, repo, prNumber, labelName)
		return err
	})
	if err != nil {
		return err
	}

	// Do not delete the label from the repo as that deletes the label from all PRs.

	delete(g.labels, labelName)
	return nil
}

func NewGitHubRequestHandler() *GithubRequestHandler {
	h, err := newGitHubRequestHandler()
	if err != nil {
		log.Fatalf("error: Could not initialise GitHub client: %v", err)
	}
	return h
}

// newGitHubRequestHandler sets up a new GithubRequestHandler struct which
// creates an oauth2 client with a GitHub access token (as specified by the
// GITHUB_ACCESS_TOKEN environment variable), and a connection to the GitHub
// API through the github.com/google/go-github/github library. It returns the
// initialised GithubRequestHandler struct, or an error as to why the
// initialisation failed.
func newGitHubRequestHandler() (*GithubRequestHandler, error) {
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
	return &GithubRequestHandler{
		// If the environment variable GITHUB_SECRET was set then we store it in
		// the struct, this is a secret that is used to calculate a hash of the
		// message so that we can validate it.
		client:      client,
		accessToken: accesstk,
		labels:      map[string]bool{},
	}, nil
}
