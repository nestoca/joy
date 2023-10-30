.PHONY: run setup build vet test test-cov clean

run:
	@echo "Run the cli using 'go run' to pass arguments: 'go run ./cmd/joy <args>'"
	@echo "Ex: 'go run ./cmd/joy help'"

setup:
	@go install go.uber.org/mock/mockgen@latest
	@go mod download

build: generate
	@go build -ldflags "-X main.version=manual-build-$(date +%F-%T)" -o ./out/joy ./cmd/joy

vet:
	@go vet ./...

generate:
	@go generate ./...

test: generate vet
	@go test -v ./...

test-cov: generate vet
	@mkdir -p ./reports
	@go test -v ./... -coverprofile ./reports/coverage.out -covermode count
	@go tool cover -func ./reports/coverage.out

clean:
	@rm -rf ./reports ./out

fmt:
	@gofumpt -w .
	@goimports --local github.com/nestoca/joy,github.com/nestoca -w .