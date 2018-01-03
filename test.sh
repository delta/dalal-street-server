#!/bin/bash
# http://stackoverflow.com/a/19134486/975271

export DALAL_ENV=Test

# Unit tests
go test -v -run="^(Test|Benchmark)[^_](.*)" ./... -args -config="$(pwd)/config.json"
code=$?
if [ $code -neq 0 ]; then
    exit $code
fi

# Integration tests

# Get db password from "Test" section of config.json
dbPass=$(egrep "Test|DbPassword" config.json \
	| grep -C1 "Test" | tail -n1 \
	| awk '{print substr($2,2,length($2)-3)}')

migrate -url mysql://root:$dbPass@/dalalstreet_test -path ./migrations up 
go test -race -v -p=1 -run="^(Test|Benchmark)_(.*)" ./... -args -config="$(pwd)/config.json"
code=$?
migrate -url mysql://root:$dbPass@/dalalstreet_test -path ./migrations down 

exit $code
