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


ROOT_DIR=/workspace
USERCONFIG_DIR=$ROOT_DIR/user-config

if [ -z $_PR_NUMBER ]; then
  echo "skipping: don't post compatibility report for push to master"
  exit 0
fi

if ! [ -s $USERCONFIG_DIR/compat-report-validators.txt ]; then
  echo "skipping: no validators to report in compatibility report"
  exit 0
fi

echo validators to be put in compability report:
cat $USERCONFIG_DIR/compat-report-validators.txt

$GOPATH/bin/post_results -validator=compat-report -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -commit-sha=$COMMIT_SHA -pr-number=$_PR_NUMBER -branch=$BRANCH_NAME
