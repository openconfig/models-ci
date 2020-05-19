#!/bin/bash

ROOT_DIR=/workspace
ZIP_FILE=$ROOT_DIR/confd.zip
RESULTSDIR=$ROOT_DIR/results/confd
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

apt install -qy unzip
unzip $ZIP_FILE -d $RESULTSDIR/confd-unzipped
find $RESULTSDIR/confd-unzipped -name 'confd-basic-*.linux.x86_64.installer.bin' -exec {} $RESULTSDIR/confd-install \;
CONFDC=$RESULTSDIR/confd-install/bin/confdc

CONFDPATH=`find $_MODEL_ROOT -type d | tr '\n' ':'`:$_MODEL_ROOT/../third_party/ietf

$CONFDC --version > $RESULTSDIR/latest-version.txt
if bash $RESULTSDIR/script.sh $CONFDC $CONFDPATH > $OUTFILE 2> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
$GOPATH/bin/post_results -validator=confd -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
