#!/bin/bash

if [ ! -f protoc-3.2.0rc2-linux-x86_64.zip ]; then
    wget --tries=3 https://github.com/google/protobuf/releases/download/v3.2.0rc2/protoc-3.2.0rc2-linux-x86_64.zip

    echo "######## Unzipping protoc compiler ##########"
    unzip protoc-3.2.0rc2-linux-x86_64.zip -d protobuf
fi

echo "######## Adding to path ##########"
export PATH=$PATH:$(pwd)/protobuf/bin

echo "######## Fetching Go dependencies ##########"
go get -u -v github.com/golang/protobuf/{proto,protoc-gen-go}
go get -u -v golang.org/x/net/context
go get -u -v google.golang.org/grpc
go get -v github.com/gemnasium/migrate
go get -v ./...

echo "########## Building proto files ###########"
bash build_proto.sh

# Get the database password
currentStage="${DALAL_ENV:-Dev}"
dbPass=$(egrep "$currentStage|DbPassword" config.json \
        | grep -C1 $currentStage | tail -n1 \
        | awk '{print substr($2,2,length($2)-3)}')

echo "######### Running migrations ##########"
migrate -url "mysql://root:$dbPass@tcp(db:3306)/dalalstreet" -path ./migrations up
