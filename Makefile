BINARY := whosup

.PHONY: build test run clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

run:
	go run . $(ARGS)

clean:
	rm -rf bin
