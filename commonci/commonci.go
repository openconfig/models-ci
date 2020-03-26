package commonci

import (
	"context"
	"errors"
	"fmt"
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
	// RootDir is the base directory of the CI, which in GCB is /workspace.
	RootDir = "/workspace"
	// ResultsDir contains all results of the CI process.
	ResultsDir = "/workspace/results"
	// ScriptFileName by convention is the script with the validator commands.
	ScriptFileName = "script.sh"
	// OutFileName by convention contains the stdout of the script file.
	OutFileName = "out"
	// FailFileName by convention contains the stderr of the script file.
	FailFileName = "fail"
)

// ValidatorResultsDir determines where a particular validator and version's
// results are
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

// StatusName determines the status description for the version of the validator.
func (v *Validator) StatusName(version string) string {
	if v == nil {
		return ""
	}
	return v.Name + version
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
			// RunBeforeApproval is ideally false here so that we can delay this long (goyang-ygot)
			// check until after the PR is approved; however, this has 2 practical problems:
			// 1. It is inconvenient to force the user to always re-invoke the build, that is to
			// run each build twice, if the changes were trivial .
			// 2. GCB can't rebuild GitHub App builds more than 3 days ago, so the current way of
			// asking users to rebuild doesn't work as it requires the user to re-invoke the build
			// less than 3 days later, which may not be the case.
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

// AddGistComment adds a comment to a gist.
func (g *GithubRequestHandler) AddGistComment(gistID, title, output string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel() // cancel context if the function returns before the timeout

	gistComment := fmt.Sprintf("# %s\n%s", title, output)

	return Retry(5, "gist comment creation", func() error {
		_, _, err := g.client.Gists.CreateComment(ctx, gistID, &github.GistComment{Body: &gistComment})
		return err
	})

	// XXX: Unfortunately check runs are currently unsupported by GCB.
	//      Keeping this code here in case GCB supports it in the future.
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
	//
	// ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel2() // cancel context if the function returns before the timeout
	//
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
// TODO(wenbli): If the RunBeforeApproval feature is used, this function should
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
