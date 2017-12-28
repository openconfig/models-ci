#!/bin/bash

# Copyright 2017 Google, Inc.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Wrapper script to generate html documentation for a specified branch
#

OC_STAGE_DIR=/home/ghci/oc-tools/oc-stage
OC_PYANG_PLUGINS=/home/ghci/oc-pyang/openconfig_pyang/plugins
DOC_OUTPUT=/opt/nginx/nginx/html/branches


if [ -z ${GITHUB_ACCESS_TOKEN} ]
then
  echo "GITHUB_ACCESS_TOKEN not set" >&2
  exit 1
fi

if [ -z ${PUSH_BRANCH} ]
then
  echo "PUSH_BRANCH variable not set" >&2
  exit 1
fi

$OC_STAGE_DIR/oc-stage.sh -r $OC_STAGE_DIR -p $OC_PYANG_PLUGINS -o $DOC_OUTPUT -b $PUSH_BRANCH
