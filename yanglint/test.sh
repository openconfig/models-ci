#!/bin/bash

ROOT_DIR=/workspace
DEB_FILE=$ROOT_DIR/libyang.deb
YANGLINT_RESULTSDIR=$ROOT_DIR/results/yanglint
OUTFILE_NAME=out
FAILFILE_NAME=fail

if ! stat $YANGLINT_RESULTSDIR; then
  exit 0
fi

apt install $DEB_FILE

# This allows downloading of the latest libyang library, but the download
# sometimes fails. Also, we don't necessarily want the user to have debug
# unrelated issues.
# echo 'deb http://download.opensuse.org/repositories/home:/liberouter/Debian_10/ /' > /etc/apt/sources.list.d/home:liberouter.list
# wget -nv https://download.opensuse.org/repositories/home:liberouter/Debian_10/Release.key -O Release.key
# apt-key add - < Release.key
# apt-get update
# apt-get install libyang

yanglint -v > $YANGLINT_RESULTSDIR/latest-version.txt
bash $YANGLINT_RESULTSDIR/script.sh > $YANGLINT_RESULTSDIR/$OUTFILE_NAME 2> $YANGLINT_RESULTSDIR/$FAILFILE_NAME
go run /go/src/github.com/openconfig/models-ci/post_results/main.go -validator=yanglint -modelRoot=$_MODEL_ROOT -repo-slug=openconfig/models -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
