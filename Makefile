.PHONY: proto

proto:
	mkdir -p gen
	protoc \
		--proto_path=proto \
		--go_out=. \
		--go_opt=module=github.com/oyavri/pi-bully \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/oyavri/pi-bully \
		proto/bully.proto

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
