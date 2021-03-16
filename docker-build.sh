#!/bin/bash
set -x
#tail -f /dev/null

wget --tries=3 https://github.com/google/protobuf/releases/download/v3.2.0rc2/protoc-3.2.0rc2-linux-x86_64.zip -O protoc-3.2.0rc2-linux-x86_64.zip

echo "######## Unzipping protoc compiler ##########"
unzip protoc-3.2.0rc2-linux-x86_64.zip -d /root/protobuf

echo "######## Fetching Go dependencies ##########"
cd ../
go get -v github.com/gemnasium/migrate
go get -u github.com/golang/protobuf/proto
go get -u github.com/golang/protobuf/protoc-gen-go
cd $GOPATH/src/github.com/golang/protobuf/protoc-gen-go/
git reset --hard ed6926b37a637426117ccab59282c3839528a700
go install github.com/golang/protobuf/protoc-gen-go
cd $GOPATH/src/github.com/delta/dalal-street-server/
go get

git submodule update --init --recursive
