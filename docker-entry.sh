#!/bin/sh

set -x

#tail -f /dev/null


wget --tries=3 https://github.com/google/protobuf/releases/download/v3.2.0rc2/protoc-3.2.0rc2-linux-x86_64.zip -O protoc-3.2.0rc2-linux-x86_64.zip

echo "######## Unzipping protoc compiler ##########"
unzip protoc-3.2.0rc2-linux-x86_64.zip -d /root/protobuf

echo "######## Fetching Go dependencies ##########"
go get -v github.com/golang/protobuf/proto
go get -v github.com/golang/protobuf/protoc-gen-go
go get -v golang.org/x/net/context
go get -v google.golang.org/grpc
go get -v github.com/gemnasium/migrate
go get -v github.com/sendgrid/sendgrid-go


echo "######## Adding to path ##########"
export PATH=$PATH:/root/protobuf/bin

# Run the migrations
echo "########## Building proto files ###########"
bash build_proto.sh
go get -v -d ./...

# Get the database password
dbPass=$(egrep "Docker|DbPassword" config.json \
        | grep -C1 "Docker" | tail -n1 \
        | awk '{print substr($2,2,length($2)-3)}')

# Wait till mysql is accepting connections
echo "################## Waiting for mysql to accept incoming connections ##################"
maxtry=3
while [ $maxtry -gt 0 ]; do
    nc -z db 3306
    isopen=$?
    if [ $isopen -eq 0 ]; then
        break
    fi
    maxtry=maxtry-1
    sleep 1
done

echo "######### Running migrations ##########"
migrate -url "mysql://root:$dbPass@tcp(db:3306)/dalalstreet_docker" -path ./migrations up

echo "################## Starting server ##################"
go run main.go
