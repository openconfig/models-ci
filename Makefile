ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

all: get-deps lint gotests clean

get-deps:
	git clone git@github.com:openconfig/models.git ${ROOT_DIR}/models

lint:
	cd ${ROOT_DIR}/models
	${ROOT_DIR}/linter/test.sh

gotests: regexp-branch regexps

regexp-branch:
	cd ${ROOT_DIR}/models && git checkout bgp-extcomm-262

regexps:
	cd gotests/regexp && $(MAKE) all

clean:
	rm -rf ${ROOT_DIR}/models
