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

# set some defaults for use with the webhook
OC_STAGE_DIR=/home/ghci/oc-tools/oc-stage
OC_PYANG_PLUGINS=/home/ghci/oc-pyang/openconfig_pyang/plugins
DOC_OUTPUT=/opt/nginx/nginx/html/branches


show_usage () {
  # print usage statement
  echo "$0 -h -s <oc-stage dir> -p <docs plugin dir> -o <output dir>" >&2
  exit 1
}

check_args () {
  # sanity check pre-requisites

  if [ -z ${GITHUB_ACCESS_TOKEN} ]
  then
    echo "GITHUB_ACCESS_TOKEN not set" >&2
    exit 1
  fi

  if [ -z ${PUSH_BRANCH} ]
  then
    echo "PUSH_BRANCH variable not set - will build all branches"
  fi

  if [ ! -d ${OC_STAGE_DIR} ]
  then
    echo "oc-tools/oc-stage directory $OC_STAGE_DIR does not exist" >&2
    exit 1
  fi

  if [ ! -d ${OC_PYANG_PLUGINS} ]
  then
    echo "oc-pyang plugins directory $OC_PYANG_PLUGINS does not exist" >&2
    exit 1
  fi

  if [ ! -d ${DOC_OUTPUT} ]
  then
    echo "output directory $DOC_OUTPUT does not exist" >&2
    exit 1
  fi

}


# options are for overriding defaults, primarily
# while testing
while getopts "hs:o:p:" opt; do
  case "${opt}" in
  h|\?) show_usage
      exit 0
      ;;
  s)  stage_dir=${OPTARG}
      OC_STAGE_DIR=$(cd $stage_dir; pwd)
      ;;
  o)  output_dir=${OPTARG}
      DOC_OUTPUT=$(cd $output_dir; pwd)
      ;;
  p)  plugin_dir=${OPTARG}
      OC_PYANG_PLUGINS=$(cd $plugin_dir; pwd)
      ;;
  *)  show_usage
      ;;
  esac
done
shift $((OPTIND-1))

check_args

if [ -z ${PUSH_BRANCH} ]
then
  $OC_STAGE_DIR/oc-stage.sh -r $OC_STAGE_DIR -p $OC_PYANG_PLUGINS -o $DOC_OUTPUT -t -g models
else
  $OC_STAGE_DIR/oc-stage.sh -r $OC_STAGE_DIR -p $OC_PYANG_PLUGINS -o $DOC_OUTPUT -b $PUSH_BRANCH -t -g models
fi
