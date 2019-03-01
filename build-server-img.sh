#!/bin/bash

wget --tries=3 https://github.com/google/protobuf/releases/download/v3.2.0rc2/protoc-3.2.0rc2-linux-x86_64.zip -O protoc-3.2.0rc2-linux-x86_64.zip

echo "######## Unzipping protoc compiler ##########"
unzip protoc-3.2.0rc2-linux-x86_64.zip -d /root/protobuf

echo "######## Fetching Go dependencies ##########"
go get -v github.com/golang/protobuf/{proto,protoc-gen-go}
go get -v golang.org/x/net/context
go get -v google.golang.org/grpc
go get -v github.com/gemnasium/migrate
go get -v github.com/sendgrid/sendgrid-go

