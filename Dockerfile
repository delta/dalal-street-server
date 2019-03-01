FROM golang:1.10

RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl \
    netcat

WORKDIR  /go/src/github.com/delta/dalal-street-server 
COPY . .

RUN ["./build-server-img.sh"]

CMD ["./docker-entry.sh"]
