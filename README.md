# Server for Dalal Street

## Setup instructions

- You must have Go [installed](https://golang.org/doc/install) and [configured](https://golang.org/doc/install#testing).
- Set your `$GOPATH` in your `.bashrc`. (Just a place where you want to keep all your Go code)
- Append `$GOPATH/bin` to your `$PATH`.
- Clone this repository.
    - `go get github.com/thakkarparth007/dalal-street-server` (**recommended**)
    - `git clone git@github.com:thakkarparth007/dalal-street-server.git` (In this case, make sure you clone the repository in `$GOPATH/src/github.com/thakkarparth007`)
- Install ***protocol buffers*** for Go. [Click here](https://github.com/golang/protobuf). Feel free to look up more about protobufs [here](https://developers.google.com/protocol-buffers/docs/gotutorial).
- Test your installation by typing `protoc --help` in your terminal.
```
cd dalal-street-server
go get -v ./...
migrate -url "mysql://root:YOUR_MYSQL_ROOT_PASSWORD@/YOUR_DATABASE_NAME" -path ./migrations up
git submodule init
git submodule update
cd socketapi
./build_proto.sh

```
- Fill in the database credentials in the `Dev` section of **config.json**.
- Run `go run main.go`

## Docker usage instructions
- Install `docker` and `docker-compose`.
- Run `cp .env.example .env`. Fill in the *DB_NAME* and *DB_PASS* in *.env*. These are the credentials for the database container.
- Use the same credentials in `Docker` section *config.json* (*DbName* and *DbPassword*) and *docker-entry.sh* (in the `migrate` command).
- Run `docker-compose up`.
- Once the containers are up, you can get shell access by using
```
docker exec -it <CONTAINER_ID> bash
```

