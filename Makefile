.PHONY: build test lint clean

build:
	go build -o shu .

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f shu
