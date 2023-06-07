run:
	@echo "Run the cli using 'go run' to pass arguments: 'go run ./cmd/joy <args>'"
	@echo "Ex: 'go run ./cmd/joy help'"

setup:
	@go mod download

build:
	@go build -o ./out/joy ./cmd/joy

vet:
	@go vet ./...

test: vet
	@go test ./...

test-cov: vet
	@mkdir -p ./reports
	@go test ./... -coverprofile ./reports/coverage.out -covermode count
	@go tool cover -func ./reports/coverage.out

clean:
	@rm -rf ./reports ./out
