# Lint go files
lint:
	golangci-lint run

# Test go files
test:
	go test -v ./... -race