VERSION=v0.1.0

build:
	@go build -ldflags="-X main.version=$(VERSION)"  .
