#!/bin/bash

ROOT_DIR=/workspace
RESULTSDIR=$ROOT_DIR/results/misc-checks
OUTFILE=$RESULTSDIR/out
FAILFILE=$RESULTSDIR/fail

if ! stat $RESULTSDIR; then
  exit 0
fi

GO111MODULE=on go get github.com/openconfig/goyang@versions-output

# all-non-empty-files.txt
find $_MODEL_ROOT -name '*.yang' > $RESULTSDIR/all-non-empty-files.txt 2>> $FAILFILE

# pr-file-parse-log
# This output is used to check for both the version update as well as build
# reachability. The latter requirement requires the script to be generated by
# cmd_gen in a per-model manner.
if bash $RESULTSDIR/script.sh > $OUTFILE 2>> $FAILFILE; then
  # Delete fail file if it's empty and the script passed.
  find $FAILFILE -size 0 -delete
fi
cat $RESULTSDIR/*.pr-file-parse-log > $RESULTSDIR/pr-file-parse-log 2>> $FAILFILE

# changed-files.txt
REPODIR=$RESULTSDIR/base_repo
git clone -b $_HEAD_BRANCH "git@github.com:$_REPO_SLUG.git" $REPODIR
cd $REPODIR
BASE_COMMIT=$(git merge-base $COMMIT_SHA origin/master)
git diff --name-only $BASE_COMMIT | grep -E '.*\.yang$' > $RESULTSDIR/changed-files.txt 2>> $OUTFILE

# master-file-parse-log
git checkout $BASE_COMMIT &> $OUTFILE
find $REPODIR -name '*.yang' | xargs $GOPATH/bin/goyang -f oc-versions -p $REPODIR > $RESULTSDIR/master-file-parse-log 2>> $FAILFILE

go run $GOPATH/src/github.com/openconfig/models-ci/post_results/main.go -validator=misc-checks -modelRoot=$_MODEL_ROOT -repo-slug=$_REPO_SLUG -pr-branch=$_HEAD_BRANCH -commit-sha=$COMMIT_SHA
