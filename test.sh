#!/bin/bash
# http://stackoverflow.com/a/19134486/975271

export DALAL_ENV=Test

# Unit tests
go test -v -run="^(Test|Benchmark)[^_](.*)" ./... -args -config="$(pwd)/config.json"

# Integration tests
#migrate -url mysql://root:@/dalalstreet_test -path ./migrations up 
#go test -race -v -p=1 -run="^(Test|Benchmark)_(.*)" ./... -args -config="$(pwd)/config.json"
#migrate -url mysql://root:@/dalalstreet_test -path ./migrations down 
