FROM golang:1.10

RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl

WORKDIR  /go/src/github.com/delta/dalal-street-server 

CMD ["./docker-entry.sh"]
