VERSION=latest

install:
	@go install -ldflags="-X main.version=$(VERSION)" .
