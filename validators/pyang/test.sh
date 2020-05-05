#!/bin/bash

# NOTE: double-hashed comments (##) are lines rendered unnecessary by the Docker
# image "gcr.io/disco-idea-817/models-ci-image" being used for pyang-all
# validation.

########################## COMMON SETUP #############################
ROOT_DIR=/workspace
MODELROOT=$ROOT_DIR/release/yang
TESTDIR=$ROOT_DIR
VENVDIR=$TESTDIR/pyangvenv
OUTFILE_NAME=out
FAILFILE_NAME=fail
SETUP_DONE=0

setup() {
  virtualenv $VENVDIR
  source $VENVDIR/bin/activate
  SETUP_DONE=1
}

teardown(){
  rm -rf $VENVDIR
  rm -rf $OCPYANG_REPO
  rm -rf $PYANGBIND_REPO
}

########################## PYANG #############################
PYANG_RESULTSDIR=$ROOT_DIR/results/pyang

# For running older versions of pyang
run-pyang-version() {
  deactivate
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
  deactivate
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

if stat $PYANG_RESULTSDIR; then
  if [ $SETUP_DONE -eq 0 ]; then
    setup
  fi

  run-pyang-head &
  for version in $@; do
    echo running extra pyang version $version
    run-pyang-version "$version" &
  done

  pip3 install pyang
  # Run latest pyang version
  {
    pyang --version > $PYANG_RESULTSDIR/latest-version.txt
    if bash $PYANG_RESULTSDIR/script.sh $VENVDIR/bin/pyang > $PYANG_RESULTSDIR/$OUTFILE_NAME 2> $PYANG_RESULTSDIR/$FAILFILE_NAME; then
      # Delete fail file if it's empty and the script passed.
      find $PYANG_RESULTSDIR/$FAILFILE_NAME -size 0 -delete
    fi
    $GOPATH/bin/post_results -validator=pyang -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
  } &
fi

########################### OC-PYANG #############################
#OCPYANG_RESULTSDIR=$ROOT_DIR/results/oc-pyang
#OCPYANG_REPO=$TESTDIR/oc-pyang-repo
#OCPYANG_DIR=$GOPATH/src/github.com/openconfig/models-ci/validators/oc-pyang
#
#if stat $OCPYANG_RESULTSDIR; then
#  if [ $SETUP_DONE -eq 0 ]; then
#    setup
#  fi
#  git clone https://github.com/openconfig/oc-pyang $OCPYANG_REPO
#
#  pip3 install --no-cache-dir -r $OCPYANG_DIR/requirements.txt
#  pip3 install enum34
#  pip3 install jinja2
#  pip3 install setuptools
#
#  # Find the directory for the openconfig linter
#  export PYTHONPATH=$OCPYANG_REPO
#  export OCPYANG_PLUGIN_DIR=$(python3 -c \
#            'import openconfig_pyang; import os; \
#             print("%s/plugins" % \
#             os.path.dirname(openconfig_pyang.__file__))')
#
#  python3 -c 'import openconfig_pyang'
#  if [ $? -ne 0 ]; then
#    echo 'could not install pyang plugin' > $OCPYANG_FAILFILE
#    teardown
#    exit 0
#  fi
#
#  pip3 install pyang
#  {
#    if bash $OCPYANG_RESULTSDIR/script.sh $VENVDIR/bin/pyang --plugindir $OCPYANG_PLUGIN_DIR > $OCPYANG_RESULTSDIR/$OUTFILE_NAME 2> $OCPYANG_RESULTSDIR/$FAILFILE_NAME; then
#      # Delete fail file if it's empty and the script passed.
#      find $OCPYANG_RESULTSDIR/$FAILFILE_NAME -size 0 -delete
#    fi
#    $GOPATH/bin/post_results -validator=oc-pyang -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
#  } &
#fi

########################## PYANGBIND #############################
#PYANGBIND_RESULTSDIR=$ROOT_DIR/results/pyangbind
#PYANGBIND_REPO=$TESTDIR/pyangbind-repo
#
#if stat $PYANGBIND_RESULTSDIR; then
#  if [ $SETUP_DONE -eq 0 ]; then
#    setup
#  fi
#  git clone https://github.com/robshakir/pyangbind $PYANGBIND_REPO
#  pip3 install --no-cache-dir -r $PYANGBIND_REPO/requirements.txt
#  pip3 install pyangbind
#  pip3 list | grep pyangbind > $PYANGBIND_RESULTSDIR/latest-version.txt
#  find $PYANGBIND_RESULTSDIR/latest-version.txt -size 0 -delete
#
#  export PYANGBIND_PLUGIN_DIR=`/usr/bin/env python3 -c \
#    'import pyangbind; import os; print ("{}/plugin".format(os.path.dirname(pyangbind.__file__)))'`
#
#  pip3 install pyang
#  {
#    if bash $PYANGBIND_RESULTSDIR/script.sh $VENVDIR/bin/pyang --plugindir $PYANGBIND_PLUGIN_DIR > $PYANGBIND_RESULTSDIR/$OUTFILE_NAME 2> $PYANGBIND_RESULTSDIR/$FAILFILE_NAME; then
#      # Delete fail file if it's empty and the script passed.
#      find $PYANGBIND_RESULTSDIR/$FAILFILE_NAME -size 0 -delete
#    fi
#    $GOPATH/bin/post_results -validator=pyangbind -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
#  } &
#fi

########################## COMMON CLEANUP #############################
wait
teardown