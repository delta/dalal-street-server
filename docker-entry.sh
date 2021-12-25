#!/bin/sh

set -x


# go get ./...
# echo "######## Go dependencies Fetched (Full) ##########"


echo "######## Git Submodules ##########"

git submodule init
git submodule update --recursive



# Run the migrations
echo "########## Building proto files ###########"
export PATH=$PATH:/root/protobuf/bin
protoc --version
bash build_proto.sh



# Wait till mysql is accepting connections
echo "################## Waiting for mysql to accept incoming connections ##################"
maxtry=3
while [ $maxtry -gt 0 ]; do
    nc -z db ${MYSQL_PORT}
    isopen=$?
    if [ $isopen -eq 0 ]; then
        echo "#### No DB :(  #########"
        break
    fi
    maxtry=maxtry-1
    sleep 1
done

echo "######### Running migrations ##########"
migrate -url "mysql://root:${MYSQL_ROOT_PASSWORD}@tcp(db:${MYSQL_PORT})/dalalstreet_docker" -path ./migrations up

echo "################## Starting server ##################"
go run main.go



echo "########## End of server ###########"
# sleep 3600

