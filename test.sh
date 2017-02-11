#!/bin/bash
# http://stackoverflow.com/a/19134486/975271

go test -v -run="^(Test|Benchmark)[^_](.*)" ./... &&
migrate -url mysql://root:@/dalalstreet_test -path ./migrations up &&
go test -race -v -p=1 -run="^(Test|Benchmark)_(.*)" ./...
