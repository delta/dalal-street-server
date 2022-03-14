#!/bin/bash

if [[ $(docker ps --filter status=running |grep dalalstreet-db) ]]
then
    # cd into the server dir
    SCRIPT=$(readlink -f "$0")
    cd "$(dirname "$SCRIPT")/../"

    mkdir -p dumps
    time=$(date +%d-%m-%Y-%H-%M-%S)

    if [ -f .env ]
    then
        export $(cat .env | xargs)
    fi
    docker exec dalalstreet-db /usr/bin/mysqldump -u root -p${DB_PASS} dalalstreet_docker > ./dumps/dump-$time.sql
    
fi
