#!/bin/bash

set -x


echo "######## Git Submodules ##########"
git submodule init
git submodule update --recursive


echo "########## Building proto files ###########"
bash build_proto.sh


echo "################## Waiting for mysql to accept incoming connections ##################"
declare -i maxtry=3
while [ $maxtry -gt 0 ]; do
    nc -z db 3306
    isopen=$?
    if [ $isopen -eq 0 ]; then
        break
    fi
    maxtry=${maxtry}-1
    sleep 1
done


echo "######### Running migrations ##########"
migrate -path ./migrations -database "mysql://root:${MYSQL_ROOT_PASSWORD}@tcp(db)/dalalstreet_docker" up


echo "################## Starting server ##################"
go run main.go
