#!/bin/bash

ROOT_DIR=/workspace
REGEXP_RESULTSDIR=$ROOT_DIR/results/regexp
OUTFILE=$REGEXP_RESULTSDIR/out
FAILFILE=$REGEXP_RESULTSDIR/fail

mkdir -p $REGEXP_RESULTSDIR
if go test -v gotests/regexp > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
go run /go/src/github.com/openconfig/models-ci/post_results/main.go -validator=regexp -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
