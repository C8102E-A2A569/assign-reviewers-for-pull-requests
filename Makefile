.PHONY: build run test docker-up docker-down load-test clean

build:
	go build -o bin/server cmd/server/main.go

run:
	go run cmd/server/main.go

test:
	go test -v ./internal/service/... -count=1

docker-up:
	docker-compose --env-file .env up --build

docker-down:
	docker-compose --env-file .env down -v

load-test: build-load-test
	./bin/loadtest

build-load-test:
	go build -o bin/loadtest loadtest/main.go

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

