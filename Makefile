PKG_NAME=github.com/SwissDataScienceCenter/renku-gateway

.PHONY: build clean lint format test

build:
	go build -o ./build/revproxy ${PKG_NAME}/cmd/revproxy
	go build -o ./build/login ${PKG_NAME}/cmd/login

clean:
	go clean
	go clean -testcache
	rm -f build/*

lint:
	golangci-lint run --config .golangci.yaml

format:
	go fmt
	golines . -w --max-len=120 --base-formatter=gofmt --ignore-generated --shorten-comments

test:
	go test -vet=all -race -cover -p 1 ./... 

openapi:
	oapi-codegen -generate types,server,spec -package login apispec.yaml > internal/login/spec.gen.go
