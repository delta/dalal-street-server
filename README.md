![CircleCI build status](https://circleci.com/gh/delta/dalal-street-server.png)
![Go Report Card](https://goreportcard.com/badge/github.com/delta/dalal-street-server)

# Server for Dalal Street

## Prerequisites

- [docker](https://docs.docker.com/engine/installation) 
- [docker-compose](https://docs.docker.com/compose/install)


## Initial Set-up

- Run ```cp .env.example .env``` and ```cp config.json.example config.json```. 
- Fill in the  **DB_PASS** in **.env** and make any additional changes if necessary.
- Use the same credentials in **Docker** section **config.json** (**DbPassword**) 


## Build instructions

- Running server

```
docker-compose up
```
- Once the containers are up, you can get shell access by using

```
docker exec -it <CONTAINER_NAME> bash
```

- To access phpMyAdmin, visit http://localhost:{PMA_PORT}/ (or http://localhost:9040/ by default)

- If changes are made to the server files, rebuild image and run server with
```
docker-compose build
docker-compose up
```
(might require sudo, depending on permissions of volume mount './docker/' )


- To view all running docker containers:
```
docker ps
```

- Server logs are present in ./dalalstreet_docker.log



### Build process break-down


#### server-setup.sh
 - Installs and sets up Protoc Buffer
 - Fetches partial go dependencies (the remaining are installed by ```go mod download```)

#### docker-entry.sh
 - Setup git submodules
 - ```build_proto.sh``` (Generate proto files - Proto files have to be built and converted to .pb.go)
 - Run migrations
 - ```go run main.go``` (Runs the main server)

<!-- 

OLD README:



![CircleCI build status](https://circleci.com/gh/delta/dalal-street-server.png)
![Go Report Card](https://goreportcard.com/badge/github.com/delta/dalal-street-server)

# Server for Dalal Street

## Prerequisites

- Go 1.16
- Protocol buffers
- MySQL

## Build instructions

- Setting up server

Refer [Setup Wiki](https://github.com/delta/dalal-street-server/wiki/Setup-Docs) for setting up Dalal-Street-Server

- Setup submodules

```
git submodule init
git submodule update --recursive
```

- Create databases and run migrations

```
mysql -u root -p -e "CREATE DATABASE dalalstreet_dev; CREATE DATABASE dalalstreet_test;"
migrate -path "./migrations" -database "mysql://root:YOUR_MYSQL_PASSWORD@/dalalstreet_dev" up
```

- Generate proto files

```
./build_proto.sh
```

- Run `cp config.json.example config.json`
- Fill in the database credentials in the `Dev` section of **config.json**.
- Run the server
  - development - Install [air](https://github.com/cosmtrek/air) for live reload
    ```bash
    air
    ```
  - production
    ```bash
    go run main.go
    ```

## Create Migrations

```
migrate create -ext sql -dir ./migrations MIGRATION_NAME
```

## Tests

- Run the test script locally before pushing commits.

```
./test.sh
```

## Docker usage instructions

- Install [docker](https://docs.docker.com/engine/installation) and [docker-compose](https://docs.docker.com/compose/install).
- Run `cp .env.example .env`. Fill in the _DB_NAME_ and _DB_PASS_ in _.env_. These are the credentials for the database container.
- Use the same credentials in `Docker` section _config.json_ (_DbName_ and _DbPassword_) and _docker-entry.sh_ (in the `migrate` command).
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

``` -->
