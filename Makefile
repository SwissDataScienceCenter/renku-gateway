PKG_NAME=github.com/SwissDataScienceCenter/renku-gateway

.PHONY: build clean tests

build: internal/login/spec.gen.go
	go mod download
	go build -o gateway $(PKG_NAME)/cmd/gateway 

clean:
	go clean
	go clean -testcache
	rm -f gateway covprofile

tests:
	go mod download
	go test -count=1 -covermode atomic -coverprofile=covprofile -vet=all -race ./...

internal/login/spec.gen.go: apispec.yaml
	oapi-codegen -generate types,server,spec -package login $< > $@ 

