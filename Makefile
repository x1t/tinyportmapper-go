.PHONY: all build test lint clean docker build-image run help

# 颜色输出
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

# 项目信息
BINARY=tinyportmapper
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -s -w

# Go 设置
GO=go
GOBIN=$(shell $(GO) env GOPATH)/bin
CGO_ENABLED=0

# 默认目标
all: help

help:
	@echo "$(GREEN)tinyportmapper-go Build System$(NC)"
	@echo ""
	@echo "用法: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  $(GREEN)build$(NC)       - 构建二进制文件"
	@echo "  $(GREEN)build-static$(NC)- 构建静态链接二进制文件"
	@echo "  $(GREEN)test$(NC)        - 运行单元测试"
	@echo "  $(GREEN)test-race$(NC)   - 运行竞态检测测试"
	@echo "  $(GREEN)lint$(NC)        - 运行代码检查"
	@echo "  $(GREEN)lint-fix$(NC)    - 运行代码检查并自动修复"
	@echo "  $(GREEN)clean$(NC)       - 清理构建产物"
	@echo "  $(GREEN)docker$(NC)      - 构建 Docker 镜像"
	@echo "  $(GREEN)run$(NC)         - 运行程序"
	@echo "  $(GREEN)help$(NC)        - 显示此帮助信息"

# 构建
build:
	@echo "$(GREEN)构建 $(BINARY)...$(NC)"
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/tinyportmapper
	@echo "$(GREEN)构建完成: $(BINARY)$(NC)"

build-static:
	@echo "$(GREEN)构建静态链接二进制文件...$(NC)"
	CGO_ENABLED=0 $(GO) build -ldflags="$(LDFLAGS) -extldflags=-static" -o $(BINARY) ./cmd/tinyportmapper
	@echo "$(GREEN)构建完成: $(BINARY) (静态链接)$(NC)"

# 测试
test:
	@echo "$(GREEN)运行测试...$(NC)"
	$(GO) test -v ./...
	@echo "$(GREEN)测试完成$(NC)"

test-race:
	@echo "$(GREEN)运行竞态检测测试...$(NC)"
	$(GO) test -v -race ./...
	@echo "$(GREEN)测试完成$(NC)"

test-coverage:
	@echo "$(GREEN)运行覆盖率测试...$(NC)"
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)覆盖率报告: coverage.html$(NC)"

# 代码检查
lint:
	@echo "$(GREEN)运行代码检查...$(NC)"
	-golangci-lint run ./...

lint-fix:
	@echo "$(GREEN)运行代码检查并自动修复...$(NC)"
	-golangci-lint run --fix ./...

# 清理
clean:
	@echo "$(GREEN)清理构建产物...$(NC)"
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	$(GO) clean -cache
	@echo "$(GREEN)清理完成$(NC)"

# Docker
docker:
	@echo "$(GREEN)构建 Docker 镜像...$(NC)"
	docker build -t $(BINARY):$(VERSION) -t $(BINARY):latest .

build-image:
	@echo "$(GREEN)构建 Docker 镜像...$(NC)"
	docker buildx build --platform linux/amd64,linux/arm64 -t $(BINARY):$(VERSION) --push .

# 运行
run: build
	@echo "$(GREEN)运行 $(BINARY)...$(NC)"
	./$(BINARY) -l 0.0.0.0:8080 -r 127.0.0.1:80 -t -u

# 安装
install: build
	@echo "$(GREEN)安装到 /usr/local/bin...$(NC)"
	install -m 755 $(BINARY) /usr/local/bin/

# 发布
release:
	@echo "$(GREEN)发布准备...$(NC)"
	@echo "请使用 goreleaser 进行发布"
	@echo "goreleaser release --rm-dist"

# 依赖
deps:
	@echo "$(GREEN)下载依赖...$(NC)"
	$(GO) mod download
	@echo "$(GREEN)依赖下载完成$(NC)"

deps-update:
	@echo "$(GREEN)更新依赖...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "$(GREEN)依赖更新完成$(NC)"

# 代码格式化
fmt:
	@echo "$(GREEN)格式化代码...$(NC)"
	$(GO) fmt ./...
	gofmt -w .

# 安全检查
security:
	@echo "$(GREEN)运行安全检查...$(NC)"
	-golangci-lint run --enable=gosec ./...

# 性能测试
bench:
	@echo "$(GREEN)运行性能测试...$(NC)"
	$(GO) test -bench=. -benchmem ./...

# 检查
check: lint test
	@echo "$(GREEN)完整检查完成$(NC)"
