#!/bin/bash

ROOT_DIR=/workspace
GOYANGYGOT_RESULTSDIR=$ROOT_DIR/results/goyang-ygot
OUTFILE_NAME=out
FAILFILE_NAME=fail

if ! stat $GOYANGYGOT_RESULTSDIR; then
  exit 0
fi

GO111MODULE=on go list -m github.com/openconfig/ygot@latest > $GOYANGYGOT_RESULTSDIR/latest-version.txt
bash $GOYANGYGOT_RESULTSDIR/script.sh > $GOYANGYGOT_RESULTSDIR/$OUTFILE_NAME 2> $GOYANGYGOT_RESULTSDIR/$FAILFILE_NAME
go run /go/src/github.com/openconfig/models-ci/post_results/main.go -validator=goyang-ygot -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
