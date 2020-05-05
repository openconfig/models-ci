#!/bin/bash

# NOTE: double-hashed comments (##) are lines rendered unnecessary by the Docker
# image "gcr.io/disco-idea-817/models-ci-image" being used for pyang-all
# validation.

########################## SETUP #############################
ROOT_DIR=/workspace
MODELROOT=$ROOT_DIR/release/yang
TESTDIR=$ROOT_DIR
VENVDIR=$TESTDIR/pyangbindvenv
RESULTSDIR=$ROOT_DIR/results/pyangbind
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

PYANGBIND_REPO=$TESTDIR/pyangbind-repo
setup() {
  virtualenv $VENVDIR
  source $VENVDIR/bin/activate

  git clone https://github.com/robshakir/pyangbind $PYANGBIND_REPO
  pip3 install --no-cache-dir -r $PYANGBIND_REPO/requirements.txt
  pip3 install pyangbind
  pip3 install pyang
}

teardown() {
  rm -rf $VENVDIR
  rm -rf $PYANGBIND_REPO
}

setup
########################## PYANGBIND #############################
pip3 list | grep pyangbind > $RESULTSDIR/latest-version.txt
find $RESULTSDIR/latest-version.txt -size 0 -delete

export PYANGBIND_PLUGIN_DIR=`/usr/bin/env python3 -c \
  'import pyangbind; import os; print ("{}/plugin".format(os.path.dirname(pyangbind.__file__)))'`
if bash $RESULTSDIR/script.sh $VENVDIR/bin/pyang --plugindir $PYANGBIND_PLUGIN_DIR > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=pyangbind -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA

########################## COMMON CLEANUP #############################
teardown
