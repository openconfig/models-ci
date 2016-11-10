#!/bin/bash

cleanup(){
  rm -rf $TESTDIR/ietf-yang $TESTDIR/pyvenv
  rm $TESTDIR/$PLUGINFILE
}

TESTDIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODELDIR=$TESTDIR/../models/yang
PLUGINFILE="openconfig_pyang-0.1.1-py2-none-any.whl"
GETURL="http://rob.sh/files/"

# This is a repo that contains the minimum required modules
# for OpenConfig build that are external dependencies.
git clone https://github.com/robshakir/yang.git $TESTDIR/ietf-yang
curl -o $TESTDIR/$PLUGINFILE $GETURL/$PLUGINFILE

# Check that we have virtualenv available
pip install --user virtualenv
virtualenv $TESTDIR/pyvenv #&>/dev/null
source $TESTDIR/pyvenv/bin/activate #&>/dev/null
pip install -r $TESTDIR/requirements.txt #&>/dev/null
pip install $TESTDIR/$PLUGINFILE #&>/dev/null

# Find the directory for the openconfig linter
export PLUGIN_DIR=$(/usr/bin/env python -c \
          'import openconfig_pyang; import os; \
           print "%s/plugins" % \
           os.path.dirname(openconfig_pyang.__file__)')

/usr/bin/env python -c 'import openconfig_pyang'
if [ $? -ne 0 ]; then
  echo "failed: couldn't install plugin"
  cleanup
  exit 127
fi

echo "running tests against modules in $MODELDIR..."
PYANGCMD="pyang --plugindir $PLUGIN_DIR -p $MODELDIR -p /tmp/ietf-yang"
PYANGCMD="$PYANGCMD --openconfig --ignore-error=OC_RELATIVE_PATH"

FAIL=0
for i in `find $MODELDIR -maxdepth 1 -mindepth 1 -type d`; do
  if [ -e $i/.BUILD ];then 
    buildstr=""
    buildfiles=$(cat $i/.BUILD | tr '\n' ' ')
    for fn in $buildfiles; do
      buildstr="$buildstr $MODELDIR/$fn"
    done

    echo -n "testing $i..."
    $PYANGCMD $buildstr &>/dev/null
    if [ $? -ne 0 ]; then
      FAIL=$((FAIL+1))
      echo "FAIL"
    else
      echo "OK"
    fi
  fi
done

cleanup

if [ $FAIL -ne 0 ]; then
  echo "failed: $FAIL tests failed"
  exit 127
fi
