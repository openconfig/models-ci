#!/bin/bash
# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


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
  pip3 install pyang
}

teardown() {
  rm -rf $VENVDIR
  rm -rf $PYANGBIND_REPO
}

setup
########################## PYANGBIND #############################
cd "${PYANGBIND_REPO}" && git rev-parse --short HEAD > $RESULTSDIR/latest-version.txt && cd -
find $RESULTSDIR/latest-version.txt -size 0 -delete

export PYTHONPATH="${PYTHONPATH}:${PYANGBIND_REPO}"
export PYANGBIND_PLUGIN_DIR="${PYANGBIND_REPO}/pyangbind/plugin"
if bash $RESULTSDIR/script.sh $VENVDIR/bin/pyang > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=pyangbind -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-number=$_PR_NUMBER -commit-sha=$COMMIT_SHA -branch=$BRANCH_NAME
BADGEFILE=$RESULTSDIR/upload-badge.sh
if stat $BADGEFILE; then
  bash $BADGEFILE
fi

########################## CLEANUP #############################
teardown
