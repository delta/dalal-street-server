#!/bin/bash

echo "######## Setting up protobuf ########"
if [ ! -f protoc-3.2.0rc2-linux-x86_64.zip ]; then
    wget --tries=3 https://github.com/google/protobuf/releases/download/v3.2.0rc2/protoc-3.2.0rc2-linux-x86_64.zip

    echo "######## Unzipping protoc compiler ##########"
    unzip protoc-3.2.0rc2-linux-x86_64.zip -d protobuf

    echo "######## Installing protoc-gen-go ########"
    go get -u -v github.com/golang/protobuf/{proto,protoc-gen-go}
    cd $GOPATH/src/github.com/golang/protobuf/protoc-gen-go/
    git reset --hard 1918e1ff6ffd2be7bed0553df8650672c3bfe80d
    go install
fi

echo "######## Adding to path ##########"
export PATH=$PATH:$(pwd)/protobuf/bin

echo "######## Fetching Go dependencies ##########"
go get -u -v golang.org/x/net/context
go get -u -v google.golang.org/grpc
go get -v github.com/gemnasium/migrate
go get -v ./...

echo "########## Building proto files ###########"
bash build_proto.sh

# Get the database password
dbPass=$(egrep "Docker|DbPassword" config.json \
        | grep -C1 "Docker" | tail -n1 \
        | awk '{print substr($2,2,length($2)-3)}')

echo "######### Running migrations ##########"
migrate -url "mysql://root:$dbPass@tcp(db:3306)/dalalstreet_docker" -path ./migrations up
