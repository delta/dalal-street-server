#!/bin/sh

if [ ! -f protoc-3.2.0rc2-linux-x86_64.zip ]; then
    wget --tries=3 https://github.com/google/protobuf/releases/download/v3.2.0rc2/protoc-3.2.0rc2-linux-x86_64.zip

    echo "######## Unzipping protoc compiler ##########"
    unzip protoc-3.2.0rc2-linux-x86_64.zip -d protobuf
fi

echo "######## Adding to path ##########"
export PATH=$PATH:$(pwd)/protobuf/bin

# Install dependencies
echo "######## Installing dependencies ######"
npm install

# Update protoc-gen
echo "######## Updating protoc-gen ########"
cd ts-protoc-gen
npm install
npm run build
cd ..

# build proto
echo "######## Building proto #########"
npm run build:proto

# build webpack
echo "######## Build webpack ##########"
npm run build:webpack

echo "######## Starting nginx server #########"
nginx -g "daemon off;"
