#!/bin/bash

mkdir tmp
git clone https://github.com/openconfig/public tmp/public
git clone https://github.com/openconfig/oc-pyang tmp/oc-pyang
mkdir static

go run ../main.go -repo_path=/home/robjs/openconfig/public \
    -ocpyang_path=tmp/oc-pyang \
    -output_path=static \
    -output_file=docs.sh \
    -output_map=sitemap.json

./docs.sh
gcloud run deploy --project=disco-idea-817 --region us-west1
rm -rf tmp
rm docs.sh
