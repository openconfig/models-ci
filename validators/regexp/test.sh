#!/bin/bash

ROOT_DIR=/workspace
RESULTSDIR=$ROOT_DIR/results/regexp
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

mkdir -p $RESULTSDIR
if go test -v gotests/regexp > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=regexp -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
