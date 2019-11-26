ROOT_DIR:=${TRAVIS_BUILD_DIR}

all: lint_html clean

lint_html:
	${ROOT_DIR}/models-ci/linter/test.sh | ${ROOT_DIR}/models-ci/linter/lint_to_md.py -m html -f /tmp/lint.out

clean:
	rm -f /tmp/lint.out
