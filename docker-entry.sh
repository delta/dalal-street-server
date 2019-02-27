#!/bin/sh

# Wait till mysql is accepting connections
echo "################## Waiting for mysql to accept incoming connections ##################"
sleep 10

# Run the migrations
echo "################## Running setup script ##################"
mkdir /root/.ssh
bash setup.sh

# Start server
go run main.go
