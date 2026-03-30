NIX_RUN = nix develop --command

.PHONY: build test lint clean

build:
	$(NIX_RUN) go build -o shu .

test:
	$(NIX_RUN) go test ./...

lint:
	$(NIX_RUN) golangci-lint run

clean:
	rm -f shu
