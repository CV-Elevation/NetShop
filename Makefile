.PHONY: proto proto-install init 

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


# 初始化目录结构
init:
	# ── 通用微服务 ──────────────────────────────────────────────────────────────
	@for svc in frontend product cart recommend ad checkout payment email user; do \
		mkdir -p $$svc/cmd \
		         $$svc/internal/handler \
		         $$svc/internal/service \
		         $$svc/internal/repository; \
		touch $$svc/cmd/main.go \
		      $$svc/internal/handler/.gitkeep \
		      $$svc/internal/service/.gitkeep \
		      $$svc/internal/repository/.gitkeep \
		      $$svc/Dockerfile \
		      $$svc/go.mod; \
	done

	# ── frontend 特殊子目录 ─────────────────────────────────────────────────────
	@mkdir -p frontend/internal/middleware \
	           frontend/internal/client
	@touch frontend/internal/middleware/.gitkeep \
	       frontend/internal/client/.gitkeep

	# ── recommend 特殊子目录 ────────────────────────────────────────────────────
	@mkdir -p recommend/internal/service/strategy
	@touch recommend/internal/service/strategy/.gitkeep

	# ── aiassistant 服务 ────────────────────────────────────────────────────────
	@mkdir -p aiassistant/cmd \
	           aiassistant/internal/handler \
	           aiassistant/internal/service/llm \
	           aiassistant/internal/service/tool \
	           aiassistant/internal/service/rag \
	           aiassistant/internal/repository
	@touch aiassistant/cmd/main.go \
	       aiassistant/internal/handler/.gitkeep \
	       aiassistant/internal/service/llm/.gitkeep \
	       aiassistant/internal/service/tool/.gitkeep \
	       aiassistant/internal/service/rag/.gitkeep \
	       aiassistant/internal/repository/.gitkeep \
	       aiassistant/Dockerfile \
	       aiassistant/go.mod

	# ── email 特殊子目录 ────────────────────────────────────────────────────────
	@mkdir -p email/internal/template
	@touch email/internal/template/.gitkeep

	# ── shared/ 共享代码 ────────────────────────────────────────────────────────
	@mkdir -p ../shared/middleware \
	           ../shared/config \
	           ../shared/pkg
	@touch ../shared/middleware/.gitkeep \
	       ../shared/config/.gitkeep \
	       ../shared/pkg/.gitkeep

	@for p in common product cart recommend ad aiassistant checkout payment email user; do \
		touch ../shared/proto/$$p.proto; \
		mkdir -p ../shared/gen/golang/$$p; \
		touch ../shared/gen/golang/$$p/.gitkeep; \
	done

	@echo "✅ 目录结构初始化完成！"
