#!/bin/bash

cleanup(){
  rm -rf $TESTDIR/pyvenv
  rm -rf $PLUGINDIR
}

escape() {
   sed \
     -e 's/\//\\\//g' \
     -e 's/\&/\\\&/g'
}


TESTDIR=$TRAVIS_BUILD_DIR/models-ci/linter
MODELROOT=$TESTDIR/../../
MODELDIR=$OCDIR
CIROOT=$TESTDIR/..
PLUGINDIR=$TESTDIR/oc-pyang

git clone https://github.com/openconfig/oc-pyang $PLUGINDIR &>/dev/null
cd $PLUGINDIR
git checkout wenbli-dev
cd -

# Check that we have virtualenv available
pip3 install --user virtualenv &>/dev/null
virtualenv $TESTDIR/pyvenv &>/dev/null
source $TESTDIR/pyvenv/bin/activate &>/dev/null
#pip3 install --user -U pip #&>/dev/null
#pip3 install --no-cache-dir -r $TESTDIR/requirements.txt #&>/dev/null
pip3 install -r $TESTDIR/requirements.txt &>/dev/null
export PYTHONPATH=$PLUGINDIR
#pip3 install enum34 #&>/dev/null
pip3 install jinja2 &>/dev/null

# Find the directory for the openconfig linter
export PLUGIN_DIR=$(/usr/bin/env python3 -c \
          'import openconfig_pyang; import os; \
           print("%s/plugins" % \
           os.path.dirname(openconfig_pyang.__file__))')

/usr/bin/env python3 -c 'import openconfig_pyang' &>/dev/null
if [ $? -ne 0 ]; then
  echo '{"status": "fail", "message": "could not install pyang plugin"}'
  cleanup
  exit 127
fi

PYANGCMD="$TESTDIR/pyvenv/bin/pyang --plugindir $PLUGIN_DIR -p $MODELDIR -p $MODELROOT/third_party/ietf-yang"
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
