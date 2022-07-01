#!/bin/bash

#
# This script orchestrates generation of oc-docs and oc-jstree documentation
# for the openconfig/public repo and deploying it directly to GCP Cloud Run.
# 

mkdir tmp
git clone https://github.com/openconfig/public tmp/public
git clone https://github.com/openconfig/oc-pyang tmp/oc-pyang
mkdir static

go run ../main.go -repo_path=tmp/public \
    -ocpyang_path=tmp/oc-pyang \
    -output_path=static \
    -output_file=docs.sh \
    -output_map=sitemap.json

python3 -m venv py
source py/bin/activate
pip install -r tmp/oc-pyang/requirements.txt
pip install --upgrade pyang
./docs.sh

# TODO(robjs): validate what GCP authentication details are needed here.
gcloud run deploy --project=disco-idea-817 --region us-west1
rm -rf py
rm -rf tmp
rm docs.sh
