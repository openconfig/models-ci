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
RESULTSDIR=$ROOT_DIR/results/goyang-ygot
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

go install github.com/openconfig/ygot/generator@latest > "${OUTFILE}" 2> "${FAILFILE}"

go list -m github.com/openconfig/ygot@latest > $RESULTSDIR/latest-version.txt
if bash $RESULTSDIR/script.sh >> $OUTFILE 2>> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=goyang-ygot -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-number=$_PR_NUMBER -commit-sha=$COMMIT_SHA -branch=$BRANCH_NAME
BADGEFILE=$RESULTSDIR/upload-badge.sh
if stat $BADGEFILE; then
  bash $BADGEFILE
fi
