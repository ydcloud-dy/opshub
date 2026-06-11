.PHONY: all build agent-binaries agent-gateway agent-gateway-binaries run clean deps swagger test fmt lint help

# 变量定义
APP_NAME=opshub
AGENT_NAME=opshub-agent
AGENT_GATEWAY_NAME=opshub-agent-gateway
BUILD_DIR=bin
AGENT_BUILD_DIR=data/agent-binaries
CONFIG_FILE=config/config.yaml
GO_FILES=$(shell find . -name '*.go' -type f)
LDFLAGS=-ldflags "-X main.Version=1.0.0 -X main.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"
AGENT_LDFLAGS=-ldflags "-X main.version=0.1.0"

# 默认目标
all: swagger build

# 编译
build:
	@echo "编译中..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) main.go
	@echo "编译完成: $(BUILD_DIR)/$(APP_NAME)"

# 编译Agent二进制
agent-binaries:
	@echo "编译 OpsHub Agent 二进制..."
	@mkdir -p $(AGENT_BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(AGENT_LDFLAGS) -o $(AGENT_BUILD_DIR)/$(AGENT_NAME)-linux-amd64 ./cmd/opshub-agent
	@GOOS=linux GOARCH=arm64 go build $(AGENT_LDFLAGS) -o $(AGENT_BUILD_DIR)/$(AGENT_NAME)-linux-arm64 ./cmd/opshub-agent
	@echo "Agent 二进制已生成到: $(AGENT_BUILD_DIR)"

# 编译 Agent Gateway
agent-gateway:
	@echo "编译 OpsHub Agent Gateway..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(AGENT_GATEWAY_NAME) ./cmd/opshub-agent-gateway
	@echo "Agent Gateway 已生成: $(BUILD_DIR)/$(AGENT_GATEWAY_NAME)"

# 编译 Agent Gateway Linux 二进制
agent-gateway-binaries:
	@echo "编译 OpsHub Agent Gateway Linux 二进制..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(AGENT_GATEWAY_NAME)-linux-amd64 ./cmd/opshub-agent-gateway
	@GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(AGENT_GATEWAY_NAME)-linux-arm64 ./cmd/opshub-agent-gateway
	@echo "Agent Gateway Linux 二进制已生成到: $(BUILD_DIR)"

# 运行服务
run:
	@echo "运行服务..."
	@go run main.go server --config $(CONFIG_FILE)

# 清理
clean:
	@echo "清理中..."
	@rm -rf $(BUILD_DIR)
	@rm -rf logs/*.log
	@echo "清理完成"

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod tidy
	@go mod verify
	@echo "依赖安装完成"

# 生成 Swagger 文档
swagger:
	@echo "生成 Swagger 文档..."
	@swag init -g main.go -o docs
	@echo "Swagger 文档生成完成"

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 格式化代码
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@echo "格式化完成"

# 代码检查
lint:
	@echo "代码检查..."
	@golangci-lint run ./...

# 帮助
help:
	@echo "可用命令:"
	@echo "  make all       - 生成 Swagger 并编译项目"
	@echo "  make build     - 编译项目"
	@echo "  make agent-binaries - 编译 Linux Agent 二进制"
	@echo "  make agent-gateway - 编译 Agent Gateway"
	@echo "  make agent-gateway-binaries - 编译 Linux Agent Gateway 二进制"
	@echo "  make run       - 运行服务"
	@echo "  make clean     - 清理编译文件和日志"
	@echo "  make deps      - 安装依赖"
	@echo "  make swagger   - 生成 Swagger 文档"
	@echo "  make test      - 运行测试"
	@echo "  make fmt       - 格式化代码"
	@echo "  make lint      - 代码检查"
