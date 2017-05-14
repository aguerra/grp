VERSION=v0.1.0

install:
	@go install -ldflags="-X main.version=$(VERSION)" .
