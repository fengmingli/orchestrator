#!/bin/bash

# Docker 构建和部署脚本
# 作者: Orchestrator Team
# 版本: v1.0
# 描述: 构建Docker镜像并推送到仓库

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 未安装，请先安装 Docker"
        exit 1
    fi
}

# 配置参数
REGISTRY=${DOCKER_REGISTRY:-""}
IMAGE_NAME=${IMAGE_NAME:-"orchestrator"}
BUILD_VERSION=${BUILD_VERSION:-"latest"}
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
BUILD_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "=== Docker 构建脚本 ==="

# 环境检查
log_info "检查Docker环境..."
check_command "docker" "Docker"

# 检查Docker是否运行
if ! docker info >/dev/null 2>&1; then
    log_error "Docker daemon 未运行，请启动Docker"
    exit 1
fi

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

log_info "项目根目录: $PROJECT_ROOT"
log_info "镜像名称: $IMAGE_NAME"
log_info "构建版本: $BUILD_VERSION"
log_info "仓库地址: ${REGISTRY:-"本地构建"}"

# 创建多阶段Dockerfile
log_info "创建Dockerfile..."
cat > Dockerfile << 'EOF'
# 多阶段构建 - 构建阶段
FROM golang:1.21-alpine AS go-builder

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制Go模块文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建Go应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags '-s -w -extldflags "-static"' \
    -o orchestrator cmd/web/main.go

# Node.js构建阶段
FROM node:18-alpine AS node-builder

WORKDIR /app

# 复制前端代码
COPY web/frontend/package*.json ./
RUN npm ci --only=production && npm cache clean --force

COPY web/frontend/ ./
RUN npm run build

# 最终运行阶段
FROM alpine:latest

# 添加非root用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata curl && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# 从构建阶段复制文件
COPY --from=go-builder /app/orchestrator .
COPY --from=node-builder /app/dist ./web/

# 设置权限
RUN chown -R appuser:appgroup /app
USER appuser

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./orchestrator"]
EOF

# 创建.dockerignore文件
log_info "创建.dockerignore..."
cat > .dockerignore << 'EOF'
# Git
.git
.gitignore

# 构建产物
bin/
build/
dist/

# 日志文件
*.log
logs/

# 依赖文件夹
node_modules/
vendor/

# 临时文件
*.tmp
*.temp

# IDE文件
.vscode/
.idea/
*.swp
*.swo

# 系统文件
.DS_Store
Thumbs.db

# 测试文件
*_test.go
test/
coverage.out

# 文档
docs/
*.md
!README.md

# 脚本
scripts/
*.sh
EOF

# 构建Docker镜像
log_info "构建Docker镜像..."

# 基础镜像标签
BASE_TAG="$IMAGE_NAME:$BUILD_VERSION"
LATEST_TAG="$IMAGE_NAME:latest"

# 如果指定了仓库地址，添加仓库前缀
if [ -n "$REGISTRY" ]; then
    BASE_TAG="$REGISTRY/$BASE_TAG"
    LATEST_TAG="$REGISTRY/$LATEST_TAG"
fi

docker build \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    --build-arg BUILD_COMMIT="$BUILD_COMMIT" \
    --build-arg BUILD_VERSION="$BUILD_VERSION" \
    -t "$BASE_TAG" \
    -t "$LATEST_TAG" \
    .

log_success "Docker镜像构建完成"

# 显示镜像信息
log_info "镜像信息:"
docker images | grep "$IMAGE_NAME" | head -5

# 镜像安全扫描 (如果可用)
if command -v docker scan &> /dev/null; then
    log_info "运行安全扫描..."
    docker scan "$BASE_TAG" || log_warning "安全扫描失败，请手动检查"
fi

# 推送到仓库 (可选)
if [ "$PUSH_IMAGE" = "true" ] && [ -n "$REGISTRY" ]; then
    log_info "推送镜像到仓库..."
    
    # 登录到仓库 (如果需要)
    if [ -n "$DOCKER_USERNAME" ] && [ -n "$DOCKER_PASSWORD" ]; then
        echo "$DOCKER_PASSWORD" | docker login "$REGISTRY" -u "$DOCKER_USERNAME" --password-stdin
    fi
    
    docker push "$BASE_TAG"
    docker push "$LATEST_TAG"
    
    log_success "镜像推送完成"
fi

# 生成docker-compose.yml示例
log_info "生成docker-compose.yml示例..."
cat > docker-compose.example.yml << EOF
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123456
      MYSQL_DATABASE: orchestrator
      MYSQL_USER: orchestrator
      MYSQL_PASSWORD: orchestrator123
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./scripts/sql:/docker-entrypoint-initdb.d
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      timeout: 20s
      retries: 10

  orchestrator:
    image: $BASE_TAG
    environment:
      DB_HOST: mysql
      DB_PORT: 3306
      DB_USERNAME: orchestrator
      DB_PASSWORD: orchestrator123
      DB_DATABASE: orchestrator
      SERVER_HOST: 0.0.0.0
      SERVER_PORT: 8080
    ports:
      - "8080:8080"
    depends_on:
      mysql:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    volumes:
      - ./logs:/app/logs

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - orchestrator
    restart: unless-stopped

volumes:
  mysql_data:
EOF

# 生成Kubernetes部署文件示例
log_info "生成Kubernetes部署文件示例..."
mkdir -p k8s
cat > k8s/deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: orchestrator
  labels:
    app: orchestrator
spec:
  replicas: 3
  selector:
    matchLabels:
      app: orchestrator
  template:
    metadata:
      labels:
        app: orchestrator
    spec:
      containers:
      - name: orchestrator
        image: $BASE_TAG
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "mysql-service"
        - name: DB_PORT
          value: "3306"
        - name: DB_USERNAME
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: password
        - name: DB_DATABASE
          value: "orchestrator"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 250m
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: orchestrator-service
spec:
  selector:
    app: orchestrator
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  type: LoadBalancer
EOF

# 清理临时文件
rm -f Dockerfile .dockerignore

echo ""
echo "==================== 构建完成 ===================="
log_success "Docker镜像: $BASE_TAG"
log_success "Latest标签: $LATEST_TAG"
echo ""
echo "📁 生成的文件:"
echo "   - docker-compose.example.yml - Docker Compose示例"
echo "   - k8s/deployment.yaml - Kubernetes部署文件"
echo ""
echo "🚀 使用说明:"
echo "   Docker运行: docker run -p 8080:8080 $BASE_TAG"
echo "   Compose启动: docker-compose -f docker-compose.example.yml up -d"
echo "   K8s部署: kubectl apply -f k8s/"
echo "=================================================="