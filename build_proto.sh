#!/bin/bash

rm -rf proto_build
mkdir -p proto_build/
cp -R proto/* proto_build/

protoc -I=proto/ --go_out=proto_build --go_opt=paths=source_relative \
--go-grpc_out=proto_build --go-grpc_opt=require_unimplemented_servers=false --go-grpc_opt=paths=source_relative proto/*.proto
protoc -I=proto/ --go_out=proto_build --go_opt=paths=source_relative proto/actions/*.proto
protoc -I=proto/ --go_out=proto_build --go_opt=paths=source_relative proto/models/*.proto
protoc -I=proto/ --go_out=proto_build --go_opt=paths=source_relative proto/datastreams/*.proto

cd proto_build
find . -type f -name "*.proto" -exec rm {} \;
go build
