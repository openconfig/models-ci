#!/bin/bash

USERCONFIG_DIR=$ROOT_DIR/user-config

if ! [ -s $USERCONFIG_DIR/compat-report-validators.txt ]; then
  # no validator needs a compability report.
  exit 0
fi

$GOPATH/bin/post_results -validator=compat-report -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
