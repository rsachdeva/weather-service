.PHONY: run run-debug run-debug-jq build-check build-test-check build fmt fmt-diff lint lint-fix fix fix-diff test update-direct-dependencies update-all-dependencies

PORT ?= 8080

run:
	go run ./cmd/weather/ -port $(PORT)

run-debug:
	LOG_LEVEL=DEBUG go run ./cmd/weather/ -port $(PORT)

## pretty-print logs: JSON lines are formatted, plain text lines pass through as-is
run-debug-jq:
	LOG_LEVEL=DEBUG go run ./cmd/weather/ -port $(PORT) 2>&1 | jq --unbuffered -R -r '. as $$line | try (fromjson | .) catch $$line'

build:
	go build -o bin/weather ./cmd/weather/

build-check:
	go build ./...

build-test-check:
	go test -run=^$ ./...

test:
	go test -v -count=1 ./...

fmt:
	go fmt ./...

fmt-diff:
	gofmt -d .

lint:
	go tool golangci-lint version
	go tool golangci-lint run

lint-fix:
	go tool golangci-lint version
	go tool golangci-lint run --fix

## go fix is the Go modernizer tool
fix-diff:
	go version
	go fix -diff ./...

fix:
	go version
	go fix ./...

update-direct-dependencies:
	go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | xargs go get -u
	go mod tidy

update-all-dependencies:
	go get -u ./...
	go mod tidy