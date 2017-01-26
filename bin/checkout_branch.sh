#!/bin/bash
THISDIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BRANCH="master"
if [ $# -eq 1 ]; then
  BRANCH=$1
fi

(
	cd $THISDIR/../models
	git checkout $BRANCH
)

