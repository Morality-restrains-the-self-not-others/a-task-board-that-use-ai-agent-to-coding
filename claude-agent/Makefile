.PHONY: help build test clean install

help:
	@echo "Available commands:"
	@echo "  build              - Build Go binary"
	@echo "  test               - Run all tests"
	@echo "  clean              - Clean up build artifacts"
	@echo "  install            - Install binary to /usr/local/bin"

build:
	./build.sh

test:
	GONOSUMDB='*' GONOSUMCHECK='*' GOPROXY=off go test ./src/... -v -count=1

clean:
	rm -rf bin/
	rm -rf .trajectories/
	go clean -cache

install: build
	cp bin/claude-agent /usr/local/bin/claude-agent
	@echo "claude-agent installed to /usr/local/bin/"
