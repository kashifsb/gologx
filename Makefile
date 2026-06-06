.PHONY: test vet lint build example clean

## Run all tests
test:
	go test -v -race -count=1 ./...

## Run go vet
vet:
	go vet ./...

## Build (compile check)
build:
	go build ./...

## Run the example
example:
	go run ./examples/

## Run tests + vet
check: vet test

## Clean build cache
clean:
	go clean -cache -testcache
