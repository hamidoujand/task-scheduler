GOLANG  := golang:1.23.0
REDIS   := redis:7.4.0
RABBITMQ := rabbitmq:3.13.6
POSTGRES := postgres:16.3
ALPINE   := alpine:3.20
VERSION  := 0.0.1
APP_NAME := tasks
IMAGE_NAME := $(APP_NAME):$(VERSION)
NAMESPACE:= tasks-system

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


### k8s
dev-apply:
	kubectl apply -f zarf/k8s/base/namespace.yml
	kubectl apply -f zarf/k8s/dev/postgres/postgres-statefulset.yml
	kubectl rollout status --namespace=$(NAMESPACE) --watch --timeout=120s sts/postgres-sts

	kubectl apply -f zarf/k8s/dev/rabbitmq/rabbitmq-statefulset.yml
	kubectl rollout status --namespace=$(NAMESPACE) --watch --timeout=120s sts/rabbitmq-sts

	kubectl apply -f zarf/k8s/dev/redis/redis-statefulset.yml	
	kubectl rollout status --namespace=$(NAMESPACE) --watch --timeout=120s sts/redis-sts

dev-delete-services:
	kubectl delete -f zarf/k8s/dev/postgres/postgres-statefulset.yml
	kubectl delete -f zarf/k8s/dev/rabbitmq/rabbitmq-statefulset.yml 
	kubectl delete -f zarf/k8s/dev/redis/redis-statefulset.yml	

	kubectl delete -f zarf/k8s/base/namespace.yml

dev-status:
	kubectl get pods -n tasks-system

run:
	SCHEDULER_API_ENVIRONMENT=development go run app/api/main.go 

tidy:
	go mod tidy 
	go mod vendor 	

help:
	go run app/api/main.go --help