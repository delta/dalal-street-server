#!/bin/bash
# http://stackoverflow.com/a/19134486/975271

export DALAL_ENV=Test

# Unit tests
# Any test that does NOT contain a "_" in its name is considered a unit test.
# It is run without initializing any tables. If you required the database to be set up
# for your test, write it as an Integration Test.
go test -v -run="^(Test|Benchmark)[^_](.*)" ./... -args -config="$(pwd)/config.json"
code=$?
if [ $code -ne 0 ]; then
    exit $code
fi

# Integration tests
printf "\nUnit Tests complete. Performing Integration Tests now.\n"

# Get db password from "Test" section of config.json
dbPass=$(egrep "Test|DbPassword" config.json \
	| grep -C1 "Test" | tail -n1 \
	| awk '{print substr($2,2,length($2)-3)}')

# Set up database
migrate -url mysql://root:$dbPass@/dalalstreet_test -path ./migrations up

# Any test that DOES contain a "_" in its name is considered an integration test.
go test -race -v -p=1 -run="^(Test|Benchmark)_(.*)" ./... -args -config="$(pwd)/config.json"
code=$?

# Tear down database
migrate -url mysql://root:$dbPass@/dalalstreet_test -path ./migrations down 

exit $code
