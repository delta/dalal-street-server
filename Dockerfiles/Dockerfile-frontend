FROM nginx:latest

RUN apt-get update && \
    apt-get install -y apt-utils \
    curl \
    git \
    vim && \
    curl -sL https://deb.nodesource.com/setup_7.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g -y webpack
