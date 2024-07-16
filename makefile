run:
	SCHEDULER_API_ENVIRONMENT=development go run app/services/scheduler/api/main.go 

tidy:
	go mod tidy 
	go mod vendor 	