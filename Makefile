.PHONY: run setup build vet test test-cov clean

run:
	@echo "Run the cli using 'go run' to pass arguments: 'go run ./cmd/joy <args>'"
	@echo "Ex: 'go run ./cmd/joy help'"

setup:
	@go install mvdan.cc/gofumpt@v0.5.0
	@go install golang.org/x/tools/cmd/goimports@v0.14.0
	@go install github.com/matryer/moq@v0.3.4
	@go mod download

build:
	@go build -ldflags "-X main.version=manual-build-$(date +%F-%T)" -o ./out/joy ./cmd/joy

vet:
	@go vet ./...

test: vet
	@go test -p 1 -v ./...

test-cov: vet
	@mkdir -p ./reports
	@go test -p 1 -v ./... -coverprofile ./reports/coverage.out -covermode count
	@go tool cover -func ./reports/coverage.out

clean:
	@rm -rf ./reports ./out

fmt:
	@gofumpt -w .
	@goimports --local github.com/nestoca/joy -w .
