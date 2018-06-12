FROM golang:1.10

LABEL maintainer Florent BERREZ

# Setup work directory
WORKDIR /go/src/github.com/fberrez/romantic-aggregator

# Install dependencies
ADD . /go/src/github.com/fberrez/romantic-aggregator/
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure
RUN go install -race -v ./...

CMD ["/go/bin/romantic-aggregator"]
