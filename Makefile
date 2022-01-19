# Work on ksh/fish

all: debug

.PHONY: tidy
tidy: clean
	go mod tidy

.PHONY: debug
debug:
	export CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ; go build
	[[ -f orasql ]] && cp orasql test/

.PHONY: release
release:
	export CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ; go build -ldflags "-s -w"
	export CGO_ENABLED=0 GOOS=windows GOARCH=amd64 ; go build -ldflags "-s -w"

.PHONY: clean
clean:
	go clean
	rm test/orasql*
