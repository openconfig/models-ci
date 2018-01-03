ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

all: get-deps lint gotests clean

get-deps:
	git clone https://${GITHUB_TOKEN}@github.com/openconfig/models.git 2>&1 >/dev/null

lint:
	cd ${ROOT_DIR}/models
	${ROOT_DIR}/linter/test.sh ${BRANCH}

lint_html:
	cd ${ROOT_DIR}/models
	${ROOT_DIR}/linter/test.sh ${BRANCH} | ${ROOT_DIR}/linter/lint_to_md.py -m html -f /tmp/lint.out

gotests: regexps

regexps:
	${ROOT_DIR}/bin/checkout_branch.sh ${BRANCH}
	cd gotests/regexp && $(MAKE) all 2>&1 >> /tmp/go-tests.out
	${ROOT_DIR}/bin/checkout_branch.sh master

clean:
	rm /tmp/lint.out; rm /tmp/go-tests.out; rm -rf ${ROOT_DIR}/models
