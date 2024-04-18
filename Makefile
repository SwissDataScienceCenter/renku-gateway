PKG_NAME=github.com/SwissDataScienceCenter/renku-gateway

.PHONY: build clean tests auth_tests run_revproxy

auth_tests:
	poetry run flake8 -v
	poetry run pytest

build: internal/login/spec.gen.go
	go mod download
	go build -o revproxy $(PKG_NAME)/cmd/revproxy 

clean:
	go clean
	go clean -testcache
	rm -f revproxy covprofile

tests:
	go mod download
	go test -count=1 -covermode atomic -coverprofile=covprofile -vet=all -race ./...

internal/login/spec.gen.go: apispec.yaml
	oapi-codegen -generate types,server,spec -package login $< > $@ 

run_revproxy:
	go run $(PKG_NAME)/cmd/revproxy
