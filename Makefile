.PHONY: kafka

GIT_TAG = $(shell git describe --always --dirty --long)
DOCKER_TAG = romantic-aggregator-$(GIT_TAG)
API_PORT = 4242
GIN_MODE = debug
KAFKA_ADDRESS = 127.0.0.1:9092
KAFKA_TOPIC = romantic-aggregator

.EXPORT_ALL_VARIABLES:
	export GIN_MODE="$(GIN_MODE)"
	export API_PORT="$(API_PORT)"
	export KAFKA_ADDRESS="$(KAFKA_ADDRESS)"
	export KAFKA_TOPIC="$(KAFKA_TOPIC)"

# Runs the project
run:
	go run --race romantic-aggregator/main.go

# Builds the Docker image
docker-build:
	docker-compose build && docker-compose up

# Runs the latest docker image pushed
docker-run:
	docker-compose up

# Starts containers which contain zookeeper and kafka
dev-start:
	docker-compose up -d zookeeper kafka

# Stops containers which contain zookeeper and kafka
dev-stop:
	docker-compose stop zookeeper kafka

# Formats the Go source code
fmt:
	go fmt ./...

test:
	go test -cover ./...
