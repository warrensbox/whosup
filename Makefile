BINARY := whosup

.PHONY: build test run clean hooks

build:
	go build -o $(BINARY) .

test:
	go test ./...

run:
	go run . $(ARGS)

clean:
	rm -rf bin

# Use the Git hooks in .githooks/ (prettier on Markdown before commit and push).
hooks:
	git config core.hooksPath .githooks
