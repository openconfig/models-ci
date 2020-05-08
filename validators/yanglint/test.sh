#!/bin/bash

ROOT_DIR=/workspace
DEB_FILE=$ROOT_DIR/libyang.deb
RESULTSDIR=$ROOT_DIR/results/yanglint
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

apt install $DEB_FILE

yanglint -v > $RESULTSDIR/latest-version.txt
if bash $RESULTSDIR/script.sh > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=yanglint -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
