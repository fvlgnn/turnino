.PHONY: build check clean dev docker-build fmt test vet

build:
	go build -o bin/turnino ./cmd/turnino

check: fmt vet test

dev:
	go run ./cmd/turnino

docker-build:
	docker build -t turnino:dev .

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -d . && exit 1)

test:
	go test -race ./...

vet:
	go vet ./...

clean:
	$(RM) bin/turnino
