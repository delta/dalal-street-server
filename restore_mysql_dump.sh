#!/bin/bash

mysqldumpfile="dumps/dump-05-01-2022-23-23-41.sql" 
# Change this to the appropriate mysqldumpfile

if [ -f .env ]
then
    export $(cat .env | xargs)
fi

docker exec -i dalalstreet-db mysql -uroot -p${DB_PASS} dalalstreet_docker < ${mysqldumpfile};