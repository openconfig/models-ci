```
go install github.com/openconfig/models-ci/openconfig-ci@latest

git clone github.com/openconfig/models-ci
cd $GOPATH/src/github.com/openconfig/models-ci/openconfig-ci
openconfig-ci diff --oldp ocdiff/testdata/yang/incl --newp ocdiff/testdata/yang/incl --oldroot ocdiff/testdata/yang/old --newroot ocdiff/testdata/yang/new
```
