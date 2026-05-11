.PHONY: proto build build-rpi-arm64

BUILD_DIR := bin
APP := pi-bully

proto:
	mkdir -p gen
	protoc \
		--proto_path=proto \
		--go_out=. \
		--go_opt=module=github.com/oyavri/pi-bully \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/oyavri/pi-bully \
		proto/bully.proto

build-rpi:
	@echo "Building $(APP) for Raspberry Pi (linux/arm64)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP)-linux-arm64 ./cmd/node

mocks:
	@echo "Generating mocks..."
	@mockgen -source=task/task.go -destination=task/mocks/mock_store.go -package=mocks
	@mockgen -source=storage/storage.go -destination=storage/mocks/mock_storage.go -package=mocks

test:
	@echo "Running tests..."
	@go test -race -tags unit -cover ./...

coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out

coverage-html:
	@echo "Generating HTML coverage report..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning up test coverage outputs..."
	@-rm -f coverage.out
	@-rm -f coverage.html
