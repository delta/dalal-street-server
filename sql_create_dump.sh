#!/bin/bash

if [[ $(docker ps --filter status=running |grep dalalstreet_db) ]]
then
    mkdir -p dumps
    time=$(date +%d-%m-%Y-%H-%M-%S)
    docker exec dalalstreet_db /usr/bin/mysqldump -u root --password=root dalalstreet_docker > ./dumps/dump-$time.sql
fi
