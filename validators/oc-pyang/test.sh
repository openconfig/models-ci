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
VENVDIR=$TESTDIR/oc-pyangvenv
RESULTSDIR=$ROOT_DIR/results/oc-pyang
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

OCPYANG_REPO=$TESTDIR/oc-pyang-repo
OCPYANG_DIR=$GOPATH/src/github.com/openconfig/models-ci/validators/oc-pyang
setup() {
  virtualenv $VENVDIR
  source $VENVDIR/bin/activate

  git clone https://github.com/openconfig/oc-pyang $OCPYANG_REPO
  pip3 install --no-cache-dir -r $OCPYANG_DIR/requirements.txt
  pip3 install --no-cache-dir -r $OCPYANG_REPO/requirements.txt
  pip3 install setuptools
  pip3 install pyang
}

teardown(){
  rm -rf $VENVDIR
  rm -rf $OCPYANG_REPO
}

setup
########################## OC-PYANG #############################
# Find the directory for the openconfig linter
export PYTHONPATH=$OCPYANG_REPO
export OCPYANG_PLUGIN_DIR=$(python3 -c \
          'import openconfig_pyang; import os; \
           print("%s/plugins" % \
           os.path.dirname(openconfig_pyang.__file__))')

python3 -c 'import openconfig_pyang'
if [ $? -ne 0 ]; then
  echo 'could not install pyang plugin' > $FAILFILE
  teardown
  exit 0
fi

if bash $RESULTSDIR/script.sh $VENVDIR/bin/pyang --plugindir $OCPYANG_PLUGIN_DIR > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=oc-pyang -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-number=$_PR_NUMBER -commit-sha=$COMMIT_SHA -branch=$BRANCH_NAME
BADGEFILE=$RESULTSDIR/upload-badge.sh
if stat $BADGEFILE; then
  bash $BADGEFILE
fi

########################## CLEANUP #############################
teardown
