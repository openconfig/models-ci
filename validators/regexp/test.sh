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
  virtualenv -p py3 $VENVDIR
  source $VENVDIR/bin/activate
  pip3 install pyang
}

teardown() {
  rm -rf $VENVDIR
}

setup

########################## regexp #############################
FAIL=0

TESTFILES_FILE="$(mktemp)"
find "$ROOT_DIR/regexp-tests" -name "*.yang" -print0 > "$TESTFILES_FILE"
echo '## RFC7950 `pattern` statement' >> $FAILFILE
XSDFAILFILE=$RESULTSDIR/xsdfail
if cat "$TESTFILES_FILE" | OCDIR=$_MODEL_ROOT xargs -0 $GOPATH/src/github.com/openconfig/pattern-regex-tests/pytests/pattern_test.sh > $OUTFILE 2> $XSDFAILFILE; then
  echo "Passed." >> $FAILFILE
else
  FAIL=1
  cat $XSDFAILFILE | while read l; do
     echo "$l" >> $FAILFILE;
   done
fi
echo "" >> $FAILFILE

echo '## `posix-pattern` statement' >> $FAILFILE
POSIXFAILFILE=$RESULTSDIR/xsdfail
if cat "$TESTFILES_FILE" | xargs -0 $GOPATH/bin/gotests -model-root=$_MODEL_ROOT >> $OUTFILE 2> $POSIXFAILFILE; then
  echo "Passed." >> $FAILFILE
else
  FAIL=1
  cat $POSIXFAILFILE | while read l; do
     echo "$l" >> $FAILFILE;
   done
fi
echo "" >> $FAILFILE

if [ $FAIL -eq 0 ]; then
  # Remove the fail file if the script passed.
  cat $FAILFILE >> $OUTFILE
  rm $FAILFILE
fi

$GOPATH/bin/post_results -validator=regexp -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-number=$_PR_NUMBER -commit-sha=$COMMIT_SHA -branch=$BRANCH_NAME
BADGEFILE=$RESULTSDIR/upload-badge.sh
if stat $BADGEFILE; then
  bash $BADGEFILE
fi

########################## CLEANUP #############################
teardown
