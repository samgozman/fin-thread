# Lint go files
lint:
	golangci-lint run

# Test go files
test:
	go test -v ./... -race

# Run the project via docker compose
run:
	docker compose up --build
