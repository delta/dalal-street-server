FROM golang:1.7
RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl \
    git && \
    curl -sL https://deb.nodesource.com/setup_7.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g -y webpack 

WORKDIR  /go/src/github.com/thakkarparth007/dalal-street-server 

CMD ["./docker-entry.sh"]


