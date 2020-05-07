#!/bin/bash

########################## COMMON SETUP #############################
ROOT_DIR=/workspace
MODELROOT=$ROOT_DIR/release/yang
TESTDIR=$ROOT_DIR
VENVDIR=$TESTDIR/pyangvenv
RESULTSDIR=$ROOT_DIR/results/pyang
OUTFILE_NAME=out
FAILFILE_NAME=fail
EXTRA_VERSIONS_FILE=$ROOT_DIR/user-config/extra-pyang-versions.txt

if ! stat $RESULTSDIR; then
  exit 0
fi

########################## PYANG #############################
# For running older versions of pyang
run-pyang-version() {
  echo running extra pyang version $1
  local RESULTSDIR=$ROOT_DIR/results/pyang$1
  local VENVDIR=$TESTDIR/pyangvenv$1
  virtualenv $VENVDIR
  source $VENVDIR/bin/activate
  pip3 install pyang==$1
  if bash $RESULTSDIR/script.sh $VENVDIR/bin/pyang > $RESULTSDIR/$OUTFILE_NAME 2> $RESULTSDIR/$FAILFILE_NAME; then
    # Delete fail file if it's empty and the script passed.
    find $RESULTSDIR/$FAILFILE_NAME -size 0 -delete
  fi
  $GOPATH/bin/post_results -validator=pyang -version=$1 -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
}

run-pyang-head() {
  echo running pyang head
  local RESULTSDIR=$ROOT_DIR/results/pyang-head
  local VENVDIR=$TESTDIR/pyangvenv-head
  virtualenv $VENVDIR
  source $VENVDIR/bin/activate
  local REPODIR=$RESULTSDIR/pyang
  git clone https://github.com/mbj4668/pyang.git $REPODIR
  cd $REPODIR
  echo "THIS IS PYTHONPATH: $PYTHONPATH" # debug
  source ./env.sh
  pip3 install --no-cache-dir -r $REPODIR/requirements.txt
  if bash $RESULTSDIR/script.sh pyang > $RESULTSDIR/$OUTFILE_NAME 2> $RESULTSDIR/$FAILFILE_NAME; then
    # Delete fail file if it's empty and the script passed.
    find $RESULTSDIR/$FAILFILE_NAME -size 0 -delete
  fi
  $GOPATH/bin/post_results -validator=pyang -version="-head" -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
}

run-pyang-head &
for version in $(< $EXTRA_VERSIONS_FILE); do
  run-pyang-version "$version" &
done

# Run latest pyang version
virtualenv $VENVDIR
source $VENVDIR/bin/activate
pip3 install pyang
pyang --version > $RESULTSDIR/latest-version.txt
if bash $RESULTSDIR/script.sh $VENVDIR/bin/pyang > $RESULTSDIR/$OUTFILE_NAME 2> $RESULTSDIR/$FAILFILE_NAME; then
  # Delete fail file if it's empty and the script passed.
  find $RESULTSDIR/$FAILFILE_NAME -size 0 -delete
fi
$GOPATH/bin/post_results -validator=pyang -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA

########################## CLEANUP #############################
wait