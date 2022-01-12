# Work on ksh/fish

all: build

.PHONY: tidy
tidy: clean
	go mod tidy

.PHONY: builds
builds:
	export CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ; go build
	export CGO_ENABLED=0 GOOS=windows GOARCH=amd64 ; go build

.PHONY: build
build:
	export CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ; go build

.PHONY: clean
clean:
	go clean
