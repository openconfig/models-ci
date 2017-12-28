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

./oc-tools/oc-stage/oc-stage.sh -r ./oc-tools/oc-stage -p ./oc-pyang/openconfig_pyang/plugins -o /opt/nginx/nginx/html/branches -b master