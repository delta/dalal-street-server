#!/bin/bash
# http://stackoverflow.com/a/19134486/975271

go test -run="^(Test|Benchmark)[^_](.*)" ./... &&
migrate -url mysql://root:@/dalalstreet_test -path ./migrations up &&
go test -p=1 -run="^(Test|Benchmark)_(.*)" ./...
