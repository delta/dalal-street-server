FROM golang:1.13

RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl \
    netcat

RUN mkdir -p /go/src/github.com/delta/dalal-street-server

WORKDIR  /go/src/github.com/delta/dalal-street-server

RUN mkdir logs

COPY . .

CMD ["./docker-entry.sh"]
