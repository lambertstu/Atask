.PHONY: test test-coverage test-unit test-security test-utils test-race

# Run all tests
test:
	go test -v ./...

# Generate coverage report
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

# Run unit tests only (exclude integration tests)
test-unit:
	go test -v -short ./...

# Run security module tests
test-security:
	go test -v ./pkg/security/...

# Run utils module tests
test-utils:
	go test -v ./pkg/utils/...

# Run with race condition detection
test-race:
	go test -race ./...

# Run specific module tests
test-engine:
	go test -v ./internal/engine/...

test-tools:
	go test -v ./internal/tools/...

test-systems:
	go test -v ./internal/systems/...

# Clean coverage files
clean-coverage:
	rm -f coverage.out coverage.html