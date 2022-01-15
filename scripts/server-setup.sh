#!/bin/bash

set -x

echo "######## Downloading protoc ZIP ##########"
PB_REL="https://github.com/protocolbuffers/protobuf/releases"
curl -LO $PB_REL/download/v3.15.8/protoc-3.15.8-linux-x86_64.zip


echo "######## Unzipping protoc compiler ##########"
unzip protoc-3.15.8-linux-x86_64.zip -d /root/protobuf


echo "######## Fetching Go dependencies ##########"
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

echo "######## Go dependencies Fetched (Partial) ##########"
