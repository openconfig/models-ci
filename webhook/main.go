package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	// TODO(robjs): Many of these options can be converted to flags for the
	// webhook binary.

	// pushCIBranches is the set of branches that CI should be run on for
	// every commit.
	pushCIBranches = []string{"master"}

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
	// docsmu is a mutex used to ensure that a single docs generation goroutine
	// runs concurrently.  This serves primarily to protect against two concurrent
	// requests for the same branch.
	docsmu sync.Mutex
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
		hashSecret:  os.Getenv("GITHUB_SECRET"),
		client:      client,
		accessToken: accesstk,
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

	http.HandleFunc("/ci/repo_push", h.pushHandler)
	http.ListenAndServe(*listenSpec, nil)
}
