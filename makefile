FRONTEND_DIR = ./web
BACKEND_DIR = .
BIN_DIR = ./bin

# Docker 镜像配置
IMAGE_PREFIX = ccr.ccs.tencentyun.com/blackai/
IMAGE_NAME = dashlyai
VERSION = $(shell cat VERSION || echo "latest")
TAG = $(VERSION)

.PHONY: all build-frontend start-backend tools clean-tools build-binary build-docker imagePush clean

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

# 编译渠道管理工具
tools:
	@echo "Building channel management tools..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/channel-health-check ./cmd/channel-health-check
	@go build -o $(BIN_DIR)/channel-batch-manager ./cmd/channel-batch-manager
	@echo "Tools built successfully:"
	@echo "  - $(BIN_DIR)/channel-health-check"
	@echo "  - $(BIN_DIR)/channel-batch-manager"

# 清理编译的工具
clean-tools:
	@echo "Cleaning tools..."
	@rm -f $(BIN_DIR)/channel-health-check
	@rm -f $(BIN_DIR)/channel-batch-manager

# 构建 Linux 二进制文件
build-binary: build-frontend
	@echo "Building Linux binary..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(VERSION)'" -o $(BIN_DIR)/new-api-linux-amd64
	@echo "Binary built successfully: $(BIN_DIR)/new-api-linux-amd64"

# 构建 Docker 镜像
build-docker:
	@echo "Building Docker image: ${IMAGE_PREFIX}${IMAGE_NAME}:${TAG}"
	docker build -t ${IMAGE_PREFIX}${IMAGE_NAME}:${TAG} .
	@echo "Image built successfully!"

# 推送镜像
imagePush: build-docker
	@echo "Pushing image: ${IMAGE_PREFIX}${IMAGE_NAME}:${TAG}"
	docker push ${IMAGE_PREFIX}${IMAGE_NAME}:${TAG}
	@echo "Image pushed successfully!"

# 清理构建产物
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(FRONTEND_DIR)/dist
	@echo "Clean complete!"
