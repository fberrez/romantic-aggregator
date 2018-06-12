.PHONY: kafka

GIT_TAG = $(shell git describe --always --dirty --long)
DOCKER_TAG = romantic-aggregator-$(GIT_TAG)

# Builds the Docker image
build:
	docker build -t "$(DOCKER_TAG)" .

# Runs the docker image previously built
run:
	docker run --net="host" -e KAFKA_ADDRESS=127.0.0.1:9092 "$(DOCKER_TAG)"

# Runs the Kafka Docker image
kafka:
	docker-compose up

# Formats the Go source code
fmt:
	go fmt ./...

test:
	go test -cover ./...
