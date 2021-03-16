FROM golang:1.13

RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl \
    netcat

ENV PATH $PATH:/root/protobuf/bin

RUN mkdir -p /go/src/github.com/delta/dalal-street-server

WORKDIR  /go/src/github.com/delta/dalal-street-server

RUN mkdir logs

COPY . .

RUN bash docker-build.sh

CMD ["./docker-entry.sh"]
