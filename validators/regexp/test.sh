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
FAIL=0

echo "## RFC7950 pattern statement" >> $FAILFILE
XSDFAILFILE=$RESULTSDIR/xsdfail
if ! OCDIR=$_MODEL_ROOT $GOPATH/src/github.com/openconfig/pattern-regex-tests/pytests/pattern_test.sh > $OUTFILE 2> $XSDFAILFILE; then
  FAIL=1
  cat $XSDFAILFILE >> $FAILFILE
fi
echo "" >> $FAILFILE

echo "## posix-pattern statement" >> $FAILFILE
POSIXFAILFILE=$RESULTSDIR/xsdfail
if ! $GOPATH/bin/gotests -model-root=$_MODEL_ROOT $GOPATH/src/github.com/openconfig/pattern-regex-tests/testdata/regexp-test.yang >> $OUTFILE 2> $POSIXFAILFILE; then
  FAIL=1
  cat $POSIXFAILFILE >> $FAILFILE
fi
echo "" >> $FAILFILE

if [ $FAIL -eq 0 ]; then
  # Delete fail file if the script passed.
  rm $FAILFILE
fi

$GOPATH/bin/post_results -validator=regexp -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-number=$_PR_NUMBER -commit-sha=$COMMIT_SHA -branch=$BRANCH_NAME
BADGEFILE=$RESULTSDIR/upload-badge.sh
if stat $BADGEFILE; then
  bash $BADGEFILE
fi

########################## CLEANUP #############################
teardown
