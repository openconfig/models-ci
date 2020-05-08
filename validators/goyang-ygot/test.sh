#!/bin/bash

ROOT_DIR=/workspace
RESULTSDIR=$ROOT_DIR/results/goyang-ygot
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

GO111MODULE=on go list -m github.com/openconfig/ygot@latest > $RESULTSDIR/latest-version.txt
if bash $RESULTSDIR/script.sh > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=goyang-ygot -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
