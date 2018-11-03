![CircleCI build status](https://circleci.com/gh/delta/dalal-street-server.png) 

# Server for Dalal Street

## Prerequisites
- Go 1.10 [Download link](https://golang.org/dl/#go1.10)
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

## Build instructions

- Download the repository and `cd` into it.
```
go get github.com/delta/dalal-street-server
cd $GOPATH/src/github.com/delta/dalal-street-server
```
- Install dependencies
```
go get -v ./...
go get -v github.com/gemnasium/migrate
go get -v gopkg.in/jarcoal/httpmock.v1
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
- Fill in the database credentials in the `Dev` section of **config.json**.
- Run `go run main.go`

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
