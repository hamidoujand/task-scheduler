GOLANG  := golang:1.23.0
REDIS   := redis:7.4.0
RABBITMQ := rabbitmq:3.13.6
POSTGRES := postgres:16.3
ALPINE   := alpine:3.20
VERSION  := 0.0.1
APP_NAME := tasks
IMAGE_NAME := $(APP_NAME):$(VERSION)
NAMESPACE:= tasks-system
TEMP_DIR  := ./temp
COMPOSE_FILE := zarf/compose/docker-compose.yml  # Path to docker-compose file

### Downlaod images 
docker-pull:
	docker pull $(GOLANG) & \
	docker pull $(ALPINE) & \
	docker pull $(POSTGRES) & \
	docker pull $(REDIS) & \
	docker pull $(RABBITMQ) & \
	wait;


### Build image
build: tasks

tasks:
	docker build \
		-f zarf/docker/tasks.dockerfile \
		-t $(IMAGE_NAME) \
		--build-arg BUILD=$(VERSION) \
		.


#===============================================================================
# Tests 
test: 
	CGO_ENABLED=0 go test -timeout=5m -count=1 ./...


up:
	mkdir -p $(TEMP_DIR)/postgres_data
	mkdir -p $(TEMP_DIR)/rabbitmq_data
	mkdir -p $(TEMP_DIR)/redis_data
	# Set the TEMP_DIR environment variable and run docker-compose up
	TEMP_DIR=$(TEMP_DIR) IMAGE_NAME=$(IMAGE_NAME) docker-compose -f $(COMPOSE_FILE) up --remove-orphans -d 

down:
	docker-compose -f $(COMPOSE_FILE) down
clean:
	rm -rf $(TEMP_DIR)

logs:
	docker-compose -f $(COMPOSE_FILE) logs -f tasks

tidy:
	go mod tidy 
	go mod vendor 	

help:
	go run app/api/main.go --help