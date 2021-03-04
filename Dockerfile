FROM golang:1.13

RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl \
    netcat

WORKDIR  /go/src/github.com/delta/dalal-street-server 
COPY . .

CMD ["./docker-entry.sh"]
