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
