#!/bin/bash

########################## SETUP #############################
ROOT_DIR=/workspace
RESULTSDIR=$ROOT_DIR/results/regexp
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail
VENVDIR=$ROOT_DIR/regexpvenv

if ! stat $RESULTSDIR; then
  exit 0
fi

setup() {
  virtualenv $VENVDIR
  source $VENVDIR/bin/activate
  pip3 install pyang
}

teardown() {
  rm -rf $VENVDIR
}

setup

########################## regexp #############################
XSDFAILFILE=$RESULTSDIR/xsdfail
if OCDIR=$_MODEL_ROOT $GOPATH/src/github.com/openconfig/pattern-regex-tests/pytests/pattern_test.sh > $OUTFILE 2> $XSDFAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $XSDFAILFILE -size 0 -delete
else
  echo "## RFC7950 pattern statement" >> $FAILFILE
  cat $FAILFILE $XSDFAILFILE > $FAILFILE
  echo "" >> $FAILFILE
fi

POSIXFAILFILE=$RESULTSDIR/xsdfail
if $GOPATH/bin/gotests -model-root=$_MODEL_ROOT $GOPATH/src/github.com/openconfig/pattern-regex-tests/testdata/regexp-test.yang >> $OUTFILE 2> $POSIXFAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
else
  echo "## posix-pattern statement" >> $FAILFILE
  cat $FAILFILE $POSIXFAILFILE > $FAILFILE
  echo "" >> $FAILFILE
fi

$GOPATH/bin/post_results -validator=regexp -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-number=$_PR_NUMBER -commit-sha=$COMMIT_SHA -branch=$BRANCH_NAME
BADGEFILE=$RESULTSDIR/upload-badge.sh
if stat $BADGEFILE; then
  bash $BADGEFILE
fi

########################## CLEANUP #############################
teardown
