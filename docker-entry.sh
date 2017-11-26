#!/bin/sh

# Wait till mysql is accepting connections
sleep 10

# Run the migrations
migrate -url "mysql://root:MYSQL_CONTAINER_ROOT_PASSWORD@tcp(db:3306)/DalalStreet" -path ./migrations up

# Start server
go run main.go
