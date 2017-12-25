#!/bin/bash

rm -rf proto_build
mkdir -p proto_build/
cp -R proto/* proto_build/
cd proto_build/
protoc --go_out=plugins=grpc,import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=pb:. *.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=actions_pb:. actions/*.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=models_pb:. models/*.proto
protoc --go_out=import_prefix=github.com/thakkarparth007/dalal-street-server/proto_build/,import_path=datastreams_pb:. datastreams/*.proto
grep -rl "proto_build" . | grep -v ".sh" | xargs sed -r -i.bak 's|github.com/thakkarparth007/dalal-street-server/proto_build/(google\|golang\|github)|\1|g'
find . -type f -name "*.bak" -exec rm {} \;
find . -type f -name "*.proto" -exec rm {} \;
go build
