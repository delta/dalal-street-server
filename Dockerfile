FROM golang:1.16.9

RUN apt-get update && \
    apt-get install -y apt-utils \
    zip \
    unzip \
    vim \
    curl \
    netcat \
    git

WORKDIR  /dalal-street-server 

COPY server-setup.sh ./
RUN "./server-setup.sh"


COPY go.mod go.sum ./

RUN go mod download

# # The above setup is run seperately earlier on, so that cache can be used 
# # when rebuilding the image, in case of any change

COPY . .

CMD ["./docker-entry.sh"]
