![CircleCI build status](https://circleci.com/gh/delta/dalal-street-server.png)
![Go Report Card](https://goreportcard.com/badge/github.com/delta/dalal-street-server)

# Server for Dalal Street

## Prerequisites
- Go 1.13 [Download link](https://golang.org/dl/#go1.13)
- Protocol buffers [Download link](https://github.com/google/protobuf/releases/download/v3.2.0rc2/protoc-3.2.0rc2-linux-x86_64.zip)
- MySQL

## Check prerequisites
- Check the go version installed.
```
go version
```
- Check protobuf installation.
```
protoc --help
```
- Setup ```SECRET_KEY``` environment variable to some string

## Build instructions

- Download the repository and `cd` into it.
```
go get github.com/delta/dalal-street-server
cd $GOPATH/src/github.com/delta/dalal-street-server
```
- Install dependencies
```
cd ../
go get -v github.com/gemnasium/migrate
go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
cd $GOPATH/src/github.com/golang/protobuf/protoc-gen-go/
git reset --hard ed6926b37a637426117ccab59282c3839528a700
go install github.com/golang/protobuf/protoc-gen-go
cd dalal-street-server/
go get
```
- Setup submodules
```
git submodule init
git submodule update --recursive
```
- Create databases and run migrations
```
mysql -u root -p -e "CREATE DATABASE dalalstreet_dev; CREATE DATABASE dalalstreet_test;"
migrate -url "mysql://root:YOUR_MYSQL_ROOT_PASSWORD@/dalalstreet_dev" -path ./migrations up
```
- Generate proto files
```
./build_proto.sh
```
- Run `cp config.json.example config.json`
- Fill in the database credentials in the `Dev` section of **config.json**.
- Run `go run main.go`

## Create Migrations
```
migrate -url "mysql://root:YOUR_MYSQL_ROOT_PASSWORD@/dalalstreet_dev" -path ./migrations create migration_file_xyz
```

## Tests
- Run the test script locally before pushing commits.
```
./test.sh
```

## Docker usage instructions
- Install [docker](https://docs.docker.com/engine/installation) and [docker-compose](https://docs.docker.com/compose/install).
- Run `cp .env.example .env`. Fill in the *DB_NAME* and *DB_PASS* in *.env*. These are the credentials for the database container.
- Use the same credentials in `Docker` section *config.json* (*DbName* and *DbPassword*) and *docker-entry.sh* (in the `migrate` command).
- Run `docker-compose up`.
- Once the containers are up, you can get shell access by using
```
docker exec -it <CONTAINER_ID> bash
```
## GoMock usage instructions
- To generate mock for a file using mockgen, place this comment after import statement
```
 //go:generate mockgen -source {YOUR_FILE_NAME}.go -destination ../mocks/{YOUR_FILE_NAME}.go -package mocks
```
- To generate mocks for all packages that has above comment

```
go generate ./...

```

- To manually generate a mock package
```
mockgen -destination=mocks/{YOUR_FILE_NAME}.go -package=mocks {PATH_TO_YOUR_FILE}

```
