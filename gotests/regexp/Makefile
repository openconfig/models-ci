ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
GOPATH=${ROOT_DIR}/../../go
GOBIN=/usr/local/go/bin
OPENCONFIGDIR=${ROOT_DIR}/../../models/yang
CMDPATH=/home/ghci/bin:/usr/local/bin:/usr/bin:/bin

all: get-deps test

get-deps:

	PATH=${CMDPATH} GOPATH=${GOPATH} ${GOBIN}/go get github.com/openconfig/goyang

test:
	GOPATH=${GOPATH} ${GOBIN}/go test --ocdir=${OPENCONFIGDIR}
