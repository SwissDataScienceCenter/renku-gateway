PKG_NAME=github.com/SwissDataScienceCenter/renku-gateway

.PHONY: build clean test

build:
	go build -o gateway github.com/SwissDataScienceCenter/renku-gateway/cmd/gateway 

clean:
	go clean
	go clean -testcache
	rm -f build/*

tests:
	go test -vet=all -race -cover -p 1 ./... 

openapi:
	oapi-codegen -generate types,server,spec -package login apispec.yaml > internal/login/spec.gen.go
