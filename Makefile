.PHONY: build test lint docker

build:
	go build -o stash -ldflags "-X main.revision=$(shell git describe --tags --always)" ./app

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run

docker:
	docker build -t stash .

run:
	go run ./app --log.enabled --log.debug
