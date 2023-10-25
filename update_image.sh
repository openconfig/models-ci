#!/bin/bash
#gcloud auth configure-docker us-west1-docker.pkg.dev
docker build -t models-ci-image .
docker tag models-ci-image us-west1-docker.pkg.dev/disco-idea-817/models-ci/models-ci-image
docker push us-west1-docker.pkg.dev/disco-idea-817/models-ci/models-ci-image
