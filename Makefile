.PHONY: generate build test lint clean help

MODULE = github.com/zhiyunliu/distributed-workflow
PROTO_DIR = proto
PROTO_OUT = proto

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

generate: ## 生成 protobuf 代码
	protoc \
		--go_out=. \
		--go_opt=Mproto/workflow.proto=github.com/zhiyunliu/distributed-workflow/proto \
		--go-grpc_out=. \
		--go-grpc_opt=Mproto/workflow.proto=github.com/zhiyunliu/distributed-workflow/proto \
		proto/workflow.proto

build: ## 编译所有包
	go build ./...

test: ## 运行单元测试
	go test -v -race -count=1 ./...

test-short: ## 运行短测试（跳过集成测试）
	go test -v -short -race -count=1 ./...

lint: ## 代码检查
	go vet ./...
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

clean: ## 清理编译产物
	go clean ./...
	rm -f coverage.out

coverage: ## 生成测试覆盖率报告
	go test -v -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

tidy: ## 整理依赖
	go mod tidy

.DEFAULT_GOAL := help
