.PHONY: proto proto-install

PROTO_DIR := shared/protos
OUT_DIR   := shared/gen

# 安装代码生成工具
proto-install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 生成所有 proto 文件
proto:
	@for f in $(PROTO_DIR)/*.proto; do \
		echo "Generating $$f ..."; \
		protoc \
			--proto_path=$(PROTO_DIR) \
			--go_out=$(OUT_DIR) \
			--go_opt=paths=source_relative \
			--go-grpc_out=$(OUT_DIR) \
			--go-grpc_opt=paths=source_relative \
			$$f; \
	done
	@echo "Done."
