#!/bin/bash

ROOT_DIR=/workspace
DEB_FILE=$ROOT_DIR/libyang.deb
YANGLINT_RESULTSDIR=$ROOT_DIR/results/yanglint
OUTFILE_NAME=out
FAILFILE_NAME=fail

if ! stat $YANGLINT_RESULTSDIR; then
  exit 0
fi

apt install $DEB_FILE

yanglint -v > $YANGLINT_RESULTSDIR/latest-version.txt
if bash $YANGLINT_RESULTSDIR/script.sh > $YANGLINT_RESULTSDIR/$OUTFILE_NAME 2> $YANGLINT_RESULTSDIR/$FAILFILE_NAME; then
  # Delete fail file if it's empty and the script passed.
  find $YANGLINT_RESULTSDIR/$FAILFILE_NAME -size 0 -delete
fi
go run /go/src/github.com/openconfig/models-ci/post_results/main.go -validator=yanglint -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
