#!/bin/sh

set -x


# go get ./...
# echo "######## Go dependencies Fetched (Full) ##########"


echo "######## Git Submodules ##########"

git submodule init
git submodule update --recursive



echo "########## Building proto files ###########"
export PATH=$PATH:/root/protobuf/bin
protoc --version
bash build_proto.sh



echo "######### Running migrations ##########"
migrate -path ./migrations -database "mysql://root:${MYSQL_ROOT_PASSWORD}@tcp(db)/dalalstreet_docker" up

echo "################## Starting server ##################"
go run main.go



echo "########## End of server ###########"
# sleep 3600

