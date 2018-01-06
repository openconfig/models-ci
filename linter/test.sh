#!/bin/bash

cleanup(){
  (cd $MODELROOT && git checkout master &>/dev/null)
  rm -rf $TESTDIR/ietf-yang $TESTDIR/pyvenv
  rm $TESTDIR/$PLUGINFILE
}

escape() {
   sed \
     -e 's/\//\\\//g' \
     -e 's/\&/\\\&/g'
}


BRANCH="master"
if [ $# -eq 1 ]; then
  BRANCH=$1
fi

TESTDIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODELROOT=$TESTDIR/../models
MODELDIR=$MODELROOT/yang
CIROOT=$TESTDIR/..
PLUGINFILE="openconfig_pyang-0.1.3-py2-none-any.whl"
GETURL="http://rob.sh/files/"

brancherr=$(
  cd $MODELROOT;
  git pull &>/dev/null;
  git checkout $BRANCH &>/dev/null;
  if [ $? -ne 0 ]; then
    echo "ERR"
  fi
  git pull &>/dev/null;
)
if [ "$brancherr" == "ERR" ]; then
  cleanup
  echo -n '{"status":"fail", "message": "could not check branch'
  echo -n "$BRANCH"
  echo 'out"}'
  exit
fi

# This is a repo that contains the minimum required modules
# for OpenConfig build that are external dependencies.
git clone https://github.com/robshakir/yang.git $TESTDIR/ietf-yang &>/dev/null
curl -o $TESTDIR/$PLUGINFILE $GETURL/$PLUGINFILE &>/dev/null

# Check that we have virtualenv available
pip install --user virtualenv &>/dev/null
virtualenv $TESTDIR/pyvenv &>/dev/null
source $TESTDIR/pyvenv/bin/activate &>/dev/null
pip install -U pip &>/dev/null
pip install --no-cache-dir -r $TESTDIR/requirements.txt &>/dev/null
pip install --no-cache-dir $TESTDIR/$PLUGINFILE &>/dev/null

# Find the directory for the openconfig linter
export PLUGIN_DIR=$(/usr/bin/env python -c \
          'import openconfig_pyang; import os; \
           print "%s/plugins" % \
           os.path.dirname(openconfig_pyang.__file__)')

/usr/bin/env python -c 'import openconfig_pyang' &>/dev/null
if [ $? -ne 0 ]; then
  echo '{"status": "fail", "message": "could not install pyang plugin"}'
  cleanup
  exit 127
fi

PYANGCMD="$TESTDIR/pyvenv/bin/pyang --plugindir $PLUGIN_DIR -p $MODELDIR -p /tmp/ietf-yang"
PYANGCMD="$PYANGCMD --openconfig --ignore-error=OC_RELATIVE_PATH"

FAIL=0
echo '{ "tests": {'
MODELDIRS=`find $MODELDIR -maxdepth 1 -mindepth 1 -type d`
MODELARR=(${MODELDIRS// / })

for i in "${!MODELARR[@]}"; do
  fn="${MODELARR[i]}"

  lastchar=","
  if [ $((i+1)) -eq ${#MODELARR[@]} ]; then
    lastchar=""
  fi

  if [ -e $fn/.spec.yml ]; then
    br=$($CIROOT/bin/find_buildfiles.py -s $fn/.spec.yml)
    if [ $? -ne 0 ]; then
      echo "\"$fn\": {\"status\": \"fail\", \"message\": \"cannot extract build file\"}$lastchar"
      continue
    fi

    BRARR=(${br//$'\n'/ })
    echo -n "\"$fn\": {\"tests\": {"
    for idx in "${!BRARR[@]}"; do
      l="${BRARR[$idx]}"
      runci=$(echo $l | awk -F"!" '{ print $1 }')
      if [ "$runci" -ne "1" ]; then
        continue
      fi

      buildstr=""
      buildfiles=$(echo $l | awk -F"!" '{ print $2 }' | tr ',' ' ')
      for fn in $buildfiles; do
        buildstr="$buildstr $MODELROOT/$fn"
      done

      modelname=$(echo $l | awk -F"!" '{ print $3 }')

      tlastchar=","
      if [ $((idx+1)) -eq ${#BRARR[@]} ]; then
        tlastchar=""
      fi

      output=$($PYANGCMD $buildstr 2>&1)
      if [ $? -ne 0 ]; then
        FAIL=$((FAIL+1))
        echo -n "\"$modelname\": {\"status\": \"fail\", \"messages\": ["
        readarray -t outputlines <<< "$output"
        for lidx in "${!outputlines[@]}"; do
          mlastchar=","
          if [ $((lidx+1)) -eq ${#outputlines[@]} ]; then
            mlastchar=""
          fi
          out=$(echo ${outputlines[$lidx]} | sed -e 's/"/\\"/g')
          echo -n "\"$out\"$mlastchar"
        done
        echo "]}$tlastchar"
      else
        echo "\"$modelname\": {\"status\": \"pass\"}$tlastchar"
      fi
    done
    echo -n "}}$lastchar"
  else
    echo "\"$fn\": {\"status\": \"fail\", \"message\": \"did not have a valid build file\"}$lastchar"
  fi
done
echo "}}"

cleanup

if [ $FAIL -ne 0 ]; then
  exit 127
fi
