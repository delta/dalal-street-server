FROM golang:1.7

RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl

WORKDIR  /go/src/github.com/thakkarparth007/dalal-street-server 

CMD ["./docker-entry.sh"]
