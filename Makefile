.PHONY: build clean deploy remove

# Build all Go binaries for Lambda
build:
	@echo "Building Go binaries..."
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/bootstrap cmd/app.go
	@echo "Build complete"


# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	-@rm -rf .serverless/
	@echo "Clean complete"

# Build and deploy to dev
deploy: build
	@echo "Deploying to dev..."
	@serverless deploy --stage dev

# Deploy to production
deploy-prod: build
	@echo "Deploying to production..."
	@serverless deploy --stage prod

# Remove deployment
remove:
	@echo "Removing deployment..."
	@serverless remove

# Local development - install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Run tests
test:
	@go test ./...

# Lint code
lint:
	@golangci-lint run

# Package only (no deploy)
package: build
	@serverless package