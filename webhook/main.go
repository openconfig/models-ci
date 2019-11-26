package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
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

	// TODO(robjs): Many of these options can be converted to flags for the
	// webhook binary.

	// repoToRunIn ensures that tests only run in the specified repository to
	// avoid false assumptions. The default for this should be openconfig/models.
	repoToRunIn = "openconfig/models"

	// pushCIBranches is the set of branches that CI should be run on for
	// every commit.
	pushCIBranches = []string{"master"}

	// goOutputPath stores the path at which the output of the Go tests is
	// stored. By default this should be /tmp/go-tests.out.
	goOutputPath = "/tmp/go-tests.out"

	// lintOutputPath stores the path at which the output of the linter is stored.
	// By default this should be /tmp/lint.out.
	lintOutputPath = "/tmp/lint.out"

	// modelsDir is the directory in which the CI testing repository is cloned on the host
	// system. The default for this value should be /home/ghci/models-ci.
	modelsDir = flag.String("mdir", "/home/ghci/models-ci", "directory where CI testing repo is cloned")

	// listenSpec is the host and port that the hook should listen on. By default
	// it should be :8080.
	listenSpec = flag.String("listen", ":8080", "host and port to listen on (<hostname>:<port>)")

	// docGenLoc is the directory where the master doc gen script can be found.  By default
	// it is in /home/ghci/models-ci/bin
	docGenLoc = flag.String("docgendir", "/home/ghci/models-ci/bin", "location of the doc gen script")

	// TODO(aashaikh): add a cmd line flag to supply parameters to the docgen script
)

// githubRequestHandler carries information relating to the GitHub session that
// is being used for the continuous integration.
type githubRequestHandler struct {
	// hashSecret is the GitHub secret that is specified with the hook, it is
	// used to validate whether the response that is received is from GitHub.
	hashSecret string
	// Client is the connection to GitHub that should be utilised.
	client *github.Client
	// accessToken is the OAuth token that should be used for interactions with
	// the GitHub API and to retrieve repo contents.
	accessToken string
	// goTestPath is the path to where the output of Go tests can be found.
	goTestPath string
	// lintTestPath is the path to where the output of the OC linter can be found.
	lintTestPath string
	// mu is a mutex used to ensure that only a single test goroutine runs
	// concurrently. Because the unit tests require access to the same checked
	// out version of the models repo, then this is the safest way to ensure
	// that we do not tread on another CI test's toes.
	mu sync.Mutex
	// docsmu is a mutex used to ensure that a single docs generation goroutine
	// runs concurrently.  This serves primarily to protect against two concurrent
	// requests for the same branch.
	docsmu sync.Mutex
}

// githubPullRequestHookInput is the JSON structure that is used as content
// when a GitHub WebHook calls the server specified in this file. Only the
// relevant fields are included for JSON unmarshalling.
type githubPullRequestHookInput struct {
	Number      int64              `json:"number"`       // Nunber of the pull request
	PullRequest *githubPullRequest `json:"pull_request"` // PullRequest contains the details of the PR.
}

// githubPullRequest is the contents of the "pull_request" object of the
// JSON document used by GitHub's when a PR change is made
type githubPullRequest struct {
	ID    int64                  `json:"id"`    // ID is the identifier for the pull request.
	State string                 `json:"state"` // State is whether the PR is open/closed etc.
	Head  *githubPullRequestHead `json:"head"`  // Head describes the top-most commit in the PR.
}

// githubPullRequestRepo contains the details of the repository with which the
// PR webhook is associated.
type githubPullRequestRepo struct {
	FullName string           `json:"full_name"` // The full name of the repo in the form user/repo.
	Name     string           `json:"name"`      // The name of the repo.
	Owner    *githubRepoOwner `json:"owner"`     // Details of the owner of the repository.
}

// githubRepoOwner provides details of the owner of the repo that is associated
// with the pull request.
type githubRepoOwner struct {
	Login string `json:"login"` // Login is the owner's GitHub username.
}

// githubPullRequestHead is the details of the Head of the repo for the PR that
// has been opened.
type githubPullRequestHead struct {
	Ref  string                 `json:"ref"`  // Ref is the reference to the Head - usually a branch.
	SHA  string                 `json:"sha"`  // SHA is the commit reference.
	Repo *githubPullRequestRepo `json:"repo"` // Repo is the repo that the commit is in.
}

// githubPushEvent decodes the interesting fields of the input JSON for a push
// event from GitHub. This is used to determine where to run CI when pushes
// are done to the master branch.
type githubPushEvent struct {
	After      string                `json:"after"`      // After is the commit ID after the push event.
	Ref        string                `json:"ref"`        // Ref is the reference to the head, supplied as a branch
	Repository *githubPushRepository `json:"repository"` // Repository is the repo that the push was associated with.
}

// githubPushRepository is the repo that a push was made to.
type githubPushRepository struct {
	Name     string `json:"name"`      // Name is the name of the repository.
	FullName string `json:"full_name"` // FullName is the full name of the repository in the form owner/reponame.
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

// decodeGitHubPullReqJSON takes an input http.Request and decodes the GitHub JSON
// document that it contains, returning an error if it is not possible.
func decodeGitHubPullReqJSON(r io.Reader) (*githubPullRequestHookInput, error) {
	// Decode the JSON document that is returned by the webhook.
	decoder := json.NewDecoder(r)

	var ghIn *githubPullRequestHookInput

	if err := decoder.Decode(&ghIn); err != nil {
		return nil, fmt.Errorf("could not decode Pull Request JSON input: %v", r)
	}
	return ghIn, nil
}

// decodeGitHubPushJSON takes an input http.Request and decodes the GitHub JSON
// document that it contains - with the format expected being that which GitHub
// sends when a push happens to a repo.
func decodeGitHubPushJSON(r io.Reader) (*githubPushEvent, error) {
	decoder := json.NewDecoder(r)

	var ghIn *githubPushEvent

	if err := decoder.Decode(&ghIn); err != nil {
		return nil, fmt.Errorf("could not decode Push JSON input: %v", r)
	}
	return ghIn, nil
}

func (g *githubRequestHandler) pushHandler(w http.ResponseWriter, r *http.Request) {
	glog.Info("Received GitHub request:  ", r)

	reqID := r.Header.Get("X-GitHub-Delivery")
	if event := r.Header.Get("X-GitHub-Event"); event != "push" {
		glog.Errorf("Not processing event %s as it is not a push, is: %s", reqID, event)
		return
	}

	pushReq, err := decodeGitHubPushJSON(r.Body)
	if err != nil {
		glog.Errorf("Could not decode JSON for push event %s, err: %v", reqID, err)
		return
	}

	if !strings.Contains(pushReq.Repository.FullName, "/") {
		glog.Errorf("Could not resolve the repository name for event %s, got: %s", reqID, pushReq.Repository.FullName)
		return
	}

	repop := strings.Split(pushReq.Repository.FullName, "/")
	if len(repop) != 2 {
		glog.Errorf("Could not determine owner and repo name for event %s, got: %v", reqID, repop)
		return
	}

	// TODO(wenbli): remove pull request functionality from webhook code.
	// repoOwner, repoName := repop[0], repop[1]

	if !strings.HasPrefix(pushReq.Ref, "refs/heads/") {
		glog.Errorf("Could not resolve the branch that the push event %s was for: %s", reqID, pushReq.Ref)
		return
	}

	refp := strings.Split(pushReq.Ref, "/")
	if len(refp) != 3 {
		glog.Errorf("Could not parse the branch the push event %s was for: %v", reqID, refp)
		return
	}
	branch := refp[2]

	//TODO(aashaikh): consider moving docs generation to another handler / path
	glog.Infof("Generating updated docs for branch %s", branch)
	go g.runGenDocs(branch)

	run := false
	for _, s := range pushCIBranches {
		if s == refp[2] {
			run = true
		}
	}

	if !run {
		glog.Infof("Not running for branch %s since it was not in the selected branches", refp[2])
		return
	}

	// TODO(wenbli): remove pull request functionality from webhook code.
	// glog.Infof("Running CI for master with ref %s", reqID)
	// go g.runCI(reqID, branch, repoOwner, repoName, pushReq.After)
}

// pullRequestHandler handles an incoming pull request event from GitHub.
// It takes an input http.ResponseWriter which is used to write to the HTTP
// client (GitHub), and a pointer to the incoming HTTP request. The relevant
// CI test is triggered, and the results posted to the GitHub pull request.
func (g *githubRequestHandler) pullRequestHandler(w http.ResponseWriter, r *http.Request) {
	glog.Info("Received GitHub request: ", r)

	reqID := r.Header.Get("X-GitHub-Delivery")
	event := r.Header.Get("X-GitHub-Event")
	sig := r.Header.Get("X-Hub-Signature")
	if sig == "" {
		glog.Errorf("Not validating request %s for event %s due to missing signature", reqID, event)
	}

	// TODO(robjs): we should use the signature that was specified to determine
	// that the input was valid.
	glog.Infof("got signature: %v", sig)

	if event != "pull_request" {
		glog.Infof("Not processing event %s as it is not a PR: %s", reqID, event)
		return
	}

	glog.Infof("processing event %s, as it is a PR", reqID)

	ghIn, err := decodeGitHubPullReqJSON(r.Body)
	defer r.Body.Close()
	if err != nil {
		glog.Errorf("Could not successfully decode input from GitHub")
		return
	}

	// Avoid trying to run CI for a repo that we don't know about.
	if ghIn.PullRequest.Head.Repo.FullName != repoToRunIn {
		glog.Errorf("Not processing %s as it is not local to the models repo - from %s.", reqID, ghIn.PullRequest.Head.Repo.FullName)
		return
	}

	// Update the status to pending so that the user can see that we have received
	// this request and are ready to run the CI.
	glog.Infof("run CI for commit %s, ref %s", ghIn.PullRequest.Head.SHA, ghIn.PullRequest.Head.Ref)
	update := &githubPRUpdate{
		Owner:       ghIn.PullRequest.Head.Repo.Owner.Login,
		Repo:        ghIn.PullRequest.Head.Repo.Name,
		Ref:         ghIn.PullRequest.Head.SHA,
		Description: "OpenConfig CI Running",
		NewStatus:   "pending",
		Context:     "OpenConfig CI",
	}
	if err := g.updatePRStatus(update); err != nil {
		glog.Errorf("couldn't update PR: %s", err)
	}

	// Launch a go routine to run the PR CI.
	go g.runCI(reqID, ghIn.PullRequest.Head.Ref, ghIn.PullRequest.Head.Repo.Owner.Login, ghIn.PullRequest.Head.Repo.Name, ghIn.PullRequest.Head.SHA)
}

// runCI is a wrapper function that runs the tests that make up the CI. It is
// designed to be called within a goroutine, but tests within it should be
// serially executed. The arguments are:
//    - runID - the unique identifier for this CI run (based on GitHub event)
//    - branch - the repo branch that CI is to be run on.
//    - user - the user that owns the repo that CI is running on.
//    - repo - the repo name that CI should be run on.
//    - SHA - the hash of the commit that is to be marked with CI results.
// Results are not returned, but rather written to a GitHub gist and the
// status fields of the relevant commit.
func (g *githubRequestHandler) runCI(runID, branch, user, repo, sha string) {
	// Lock the mutex that we use to ensure a single test runs each time. Note
	// that sync.Mutex.Lock() is blocking, so we essentially just spinlock
	// until such time as we can acquire the lock.
	g.mu.Lock()
	g.runLintGoTests(runID, branch, user, repo, sha)
	// Done with tests, unlock the mutex.
	defer g.mu.Unlock()
}

// runLintGoTests runs the OpenConfig linter, and Go-based tests for the models
// repo. The results are written to a GitHub Gist, and into the PR that was
// modified, associated with the commit reference SHA.
func (g *githubRequestHandler) runLintGoTests(runID, branch, user, repo, sha string) {

	// Run the tests using exec. Env variables are set for the branch that should
	// be tested and the GitHub token.
	lintCmd := exec.Command("make", "clean", "get-deps", "lint_html")
	lintCmd.Dir = *modelsDir
	envs := []string{
		fmt.Sprintf("GITHUB_TOKEN=%s", g.accessToken),
		fmt.Sprintf("BRANCH=%s", branch),
	}
	lintCmd.Env = envs

	out, ciErr := lintCmd.CombinedOutput()
	glog.Infof("Lint test output: %s", out)

	goCmd := exec.Command("make", "gotests")
	goCmd.Dir = *modelsDir
	goCmd.Env = envs

	goout, goErr := goCmd.CombinedOutput()
	glog.Infof("Go test output: %s", goout)

	lintOK := true
	if ciErr != nil {
		lintOK = false
	}
	goOK := true
	if goErr != nil {
		goOK = false
	}

	output := fmt.Sprintf("%s\n\n%s", string(out), string(goout))
	url, _, err := g.createCIOutputGist(runID, output, lintOK, goOK)
	if err != nil {
		glog.Errorf("couldn't create gist: %s", err)
	}

	prUpdate := &githubPRUpdate{
		Owner:   user,
		Repo:    repo,
		Ref:     sha,
		URL:     url,
		Context: "OpenConfig CI",
	}

	if ciErr != nil || goErr != nil {
		prUpdate.NewStatus = "failure"
		prUpdate.Description = "OpenConfig CI Failed"

		if uperr := g.updatePRStatus(prUpdate); uperr != nil {
			glog.Errorf("couldn't update PR to failed: %s", out)
		}
		return
	}

	prUpdate.NewStatus = "success"
	prUpdate.Description = "OpenConfig CI Succeeded"
	if uperr := g.updatePRStatus(prUpdate); uperr != nil {
		glog.Errorf("couldn't update PR to succeeded: %s", uperr)
	}
}

// runGenDocs is a wrapper script that calls the docs generation
// scripts within a mutex lock.
func (g *githubRequestHandler) runGenDocs(branch string) {
	g.docsmu.Lock()
	g.generateDocs(branch)
	defer g.docsmu.Unlock()
}

// generateDocs runs the documentation generation plugin for the
// branch specified in the push request.
func (g *githubRequestHandler) generateDocs(branch string) {

	scriptfile := *docGenLoc + "/gen_docs_branch.sh"
	if _, err := os.Stat(scriptfile); err != nil {
		glog.Errorf("Doc gen script not accessible at %s: %s", scriptfile, err)
		return
	}
	docsCmd := exec.Command(scriptfile)
	envs := []string{
		fmt.Sprintf("GITHUB_ACCESS_TOKEN=%s", g.accessToken),
		fmt.Sprintf("PUSH_BRANCH=%s", branch),
	}
	docsCmd.Env = envs

	out, docsErr := docsCmd.CombinedOutput()
	glog.Infof("Doc gen output: %s", out)

	if docsErr != nil {
		glog.Errorf("Doc gen failed: %s", docsErr)
		return
	}

}

// createCIOutputGist creates a GitHub Gist, and appends comment output to it.
// In this case, the runID is used as the title for the Gist (to identify the
// changes), output is the stdout/stderr output of the CI test, and success
// indicates whether it was a successful test. The output of the /tmp/lint.out
// file is taken and this is posted as a Gist comment, along with the contents
// of the /tmp/go-tests.out file which contains other unit tests. The function
// returns the URL and ID of the Gist that was created, its ID or the error
// experienced during processing.
func (g *githubRequestHandler) createCIOutputGist(runID, output string, lintOK, goOK bool) (string, string, error) {

	d := fmt.Sprintf("OpenConfig CI Test Run Output: %s", runID)
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

	goTestOut, err := ioutil.ReadFile(g.goTestPath)
	if err != nil {
		return "", "", err
	}

	goSymbol := ":white_check_mark:"
	if !goOK {
		goSymbol = ":no_entry:"
	}
	goOut := fmt.Sprintf("```\n%s\n```", goTestOut)
	x := fmt.Sprintf("# %s Go Tests\n%s", goSymbol, goOut)
	if _, _, err = g.client.Gists.CreateComment(ctx, *gisto.ID, &github.GistComment{Body: &x}); err != nil {
		return "", "", err
	}

	return *gisto.HTMLURL, *gisto.ID, nil
}

// updatePRStatus takes an input githubPRUpdate struct and updates a GitHub
// pull request's status with the relevant details. It returns an error if
// the update was not successful.
func (g *githubRequestHandler) updatePRStatus(update *githubPRUpdate) error {
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

// newGitHubRequestHandler sets up a new githubRequestHandler struct which
// creates an oauth2 client with a GitHub access token (as specified by the
// GITHUB_ACCESS_TOKEN environment variable), and a connection to the GitHub
// API through the github.com/google/go-github/github library. It returns the
// initialised githubRequestHandler struct, or an error as to why the
// initialisation failed.
func newGitHubRequestHandler() (*githubRequestHandler, error) {
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
	return &githubRequestHandler{
		// If the environment variable GITHUB_SECRET was set then we store it in
		// the struct, this is a secret that is used to calculate a hash of the
		// message so that we can validate it.
		hashSecret:   os.Getenv("GITHUB_SECRET"),
		client:       client,
		accessToken:  accesstk,
		goTestPath:   goOutputPath,
		lintTestPath: lintOutputPath,
	}, nil
}

func main() {
	flag.Parse()

	h, err := newGitHubRequestHandler()
	if err != nil {
		glog.Errorf("Could not initialise GitHub client: %v", err)
		return
	}

	if h.hashSecret == "" {
		glog.Warning("Will not validate GitHub messages...")
	}

	// We only handle a single URL currently, which is a path for the
	// continuous integration tests.
	// TODO(wenbli): remove pull request functionality from webhook code.
	// http.HandleFunc("/ci/pull_request", h.pullRequestHandler)
	http.HandleFunc("/ci/repo_push", h.pushHandler)
	http.ListenAndServe(*listenSpec, nil)
}
