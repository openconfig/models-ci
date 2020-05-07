#!/bin/bash

ROOT_DIR=/workspace
USERCONFIG_DIR=$ROOT_DIR/user-config

echo validators to be put in compability report:
cat $USERCONFIG_DIR/compat-report-validators.txt

if ! [ -s $USERCONFIG_DIR/compat-report-validators.txt ]; then
  exit 0
fi

$GOPATH/bin/post_results -validator=compat-report -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA -pr-number=$_PR_NUMBER
