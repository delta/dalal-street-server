#!/bin/bash

mkdir -p proto_build/
cp -R proto/* proto_build/
cd proto_build/
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/,import_path=socketapi:. *.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/,import_path=socketapi/actions:. actions/*.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/,import_path=socketapi/models:. models/*.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/,import_path=socketapi/errors:. errors/*.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/,import_path=socketapi/datastreams:. datastreams/*.proto
grep -rl "github.com/golang/protobuf/proto" . | grep -v ".sh" | xargs sed -i'' 's|github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/github.com/golang/protobuf/proto|github.com/golang/protobuf/proto|g'
find . -type f -name "*.proto" -exec rm {} \;
go build

