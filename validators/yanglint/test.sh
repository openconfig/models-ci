#!/bin/bash

ROOT_DIR=/workspace
RESULTSDIR=$ROOT_DIR/results/yanglint
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

git clone https://github.com/CESNET/libyang.git libyang
cd libyang
latest=$(git tag -l | sort -V | tail -1)
git checkout $latest
mkdir build; cd build
cmake ..
make
make install

yanglint -v > $RESULTSDIR/latest-version.txt
if bash $RESULTSDIR/script.sh > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=yanglint -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-number=$_PR_NUMBER -commit-sha=$COMMIT_SHA -branch=$BRANCH_NAME
BADGEFILE=$RESULTSDIR/upload-badge.sh
if stat $BADGEFILE; then
  bash $BADGEFILE
fi
