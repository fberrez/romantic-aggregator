.PHONY: kafka

GIT_TAG = $(shell git describe --always --dirty --long)
DOCKER_TAG = romantic-aggregator-$(GIT_TAG)
API_PORT = 4242
GIN_MODE = release
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
	docker build -t "$(DOCKER_TAG)" .

# Runs the docker image previously built
docker-run:
	docker run --net="host" -e KAFKA_ADDRESS="$(KAFKA_ADDRESS)" -e API_PORT="$(API_PORT)" -e GIN_MODE="$(GIN_MODE)" "$(DOCKER_TAG)"

# Runs the Kafka Docker image
kafka:
	docker-compose up

# Formats the Go source code
fmt:
	go fmt ./...

test:
	go test -cover ./...
