TEMPL := $(shell go env GOPATH)/bin/templ$(shell go env GOEXE)

.PHONY: generate test vet build dev migrate

generate:
	$(TEMPL) generate

test: generate
	go test ./...

vet: generate
	go vet ./...

build: generate
	go build -o bin/web$(shell go env GOEXE) ./cmd/web

dev: generate
	go run ./cmd/web

migrate:
	goose -dir ./migrations mysql "$${DB_USER}:$${DB_PASSWORD}@tcp($${DB_HOST}:$${DB_PORT})/$${DB_NAME}?parseTime=true" up
