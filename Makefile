

up-depedences:
	docker-compose up -d

run-payment-api:
	go run payment/main.go

run-auth-api:
	go run authorization/main.go

run-faker:
	go run faker/main.go