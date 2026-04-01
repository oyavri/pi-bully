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
	mockgen -source=task/task.go -destination=task/mocks/mock_store.go -package=mocks
	mockgen -source=storage/storage.go -destination=storage/mocks/mock_storage.go -package=mocks
