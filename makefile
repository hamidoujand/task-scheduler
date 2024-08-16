run:
	SCHEDULER_API_ENVIRONMENT=development go run app/api/main.go 

tidy:
	go mod tidy 
	go mod vendor 	

help:
	go run app/api/main.go --help