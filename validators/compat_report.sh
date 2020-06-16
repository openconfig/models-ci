#!/bin/bash

ROOT_DIR=/workspace
USERCONFIG_DIR=$ROOT_DIR/user-config

if ! [ -s $USERCONFIG_DIR/compat-report-validators.txt ]; then
  echo "skipping: no validators to report in compatibility report"
  exit 0
fi

echo validators to be put in compability report:
cat $USERCONFIG_DIR/compat-report-validators.txt

$GOPATH/bin/post_results -validator=compat-report -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -commit-sha=$COMMIT_SHA -pr-number=$_PR_NUMBER -branch=$BRANCH_NAME
