#!/bin/bash

rm -rf proto_build
mkdir -p proto_build/
cp -R proto/* proto_build/
cd proto_build/
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=pb:. *.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=actions_pb:. actions/*.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=models_pb:. models/*.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=datastreams_pb:. datastreams/*.proto
grep -rl "github.com/golang/protobuf/proto" . | grep -v ".sh" | xargs sed -i'' 's|github.com/thakkarparth007/dalal-street-server/proto_build/github.com/golang/protobuf/proto|github.com/golang/protobuf/proto|g'
find . -type f -name "*.proto" -exec rm {} \;
go build
