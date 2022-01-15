#!/bin/bash

set -x


echo "########## Building proto files ###########"
bash ./scripts/build_proto.sh


echo "################## Waiting for mysql to accept incoming connections ##################"
until nc -z -v -w30 db 3306
do
  echo "     Waiting for database connection    "
  sleep 3
done

echo "######### Running migrations ##########"
migrate -path ./migrations -database "mysql://root:${DB_PASS}@tcp(db)/${DB_NAME}" up


echo "################## Starting server ##################"
go run main.go
