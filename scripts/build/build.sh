#!/bin/bash

# 工作流编排系统构建脚本
# 作者: Orchestrator Team
# 版本: v1.0
# 描述: 构建生产环境的前后端代码

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
        log_error "$1 未安装，请先安装 $2"
        exit 1
    fi
}

# 构建信息
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
BUILD_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
BUILD_VERSION=${BUILD_VERSION:-"v1.0.0"}

echo "=== 工作流编排系统构建脚本 ==="

# 环境检查
log_info "检查构建环境..."
check_command "go" "Go 1.21+"
check_command "node" "Node.js 18+"
check_command "npm" "npm"

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

log_info "项目根目录: $PROJECT_ROOT"
log_info "构建版本: $BUILD_VERSION"
log_info "构建时间: $BUILD_TIME"
log_info "构建分支: $BUILD_BRANCH"
log_info "构建提交: $BUILD_COMMIT"

# 清理旧的构建文件
log_info "清理旧的构建文件..."
rm -rf bin/
rm -rf web/frontend/dist/
rm -rf build/
mkdir -p bin build

# 构建后端
log_info "构建后端服务..."

# Go构建参数
LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X 'main.BuildTime=$BUILD_TIME'"
LDFLAGS="$LDFLAGS -X 'main.BuildCommit=$BUILD_COMMIT'"
LDFLAGS="$LDFLAGS -X 'main.BuildVersion=$BUILD_VERSION'"

# 构建Linux版本 (生产环境)
log_info "构建 Linux x86_64 版本..."
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o bin/orchestrator-linux-amd64 cmd/web/main.go

# 构建当前平台版本
log_info "构建当前平台版本..."
go build -ldflags "$LDFLAGS" -o bin/orchestrator cmd/web/main.go

# 构建其他平台版本 (可选)
if [ "$BUILD_ALL_PLATFORMS" = "true" ]; then
    log_info "构建多平台版本..."
    
    # Windows
    GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o bin/orchestrator-windows-amd64.exe cmd/web/main.go
    
    # macOS Intel
    GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o bin/orchestrator-darwin-amd64 cmd/web/main.go
    
    # macOS Apple Silicon
    GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o bin/orchestrator-darwin-arm64 cmd/web/main.go
    
    log_success "多平台构建完成"
fi

log_success "后端构建完成"

# 构建前端
log_info "构建前端应用..."
cd web/frontend

# 安装依赖
if [ ! -d "node_modules" ]; then
    log_info "安装前端依赖..."
    npm ci
fi

# 构建生产版本
log_info "构建前端生产版本..."
npm run build

if [ ! -d "dist" ]; then
    log_error "前端构建失败，dist目录不存在"
    exit 1
fi

log_success "前端构建完成"

cd "$PROJECT_ROOT"

# 打包构建产物
log_info "打包构建产物..."

# 创建发布目录
RELEASE_DIR="build/orchestrator-$BUILD_VERSION"
mkdir -p "$RELEASE_DIR"

# 复制后端二进制文件
cp bin/orchestrator-linux-amd64 "$RELEASE_DIR/"
cp bin/orchestrator "$RELEASE_DIR/"

# 复制前端文件
cp -r web/frontend/dist "$RELEASE_DIR/web"

# 复制配置文件和脚本
cp -r scripts "$RELEASE_DIR/"
cp README.md "$RELEASE_DIR/"
cp CLAUDE.md "$RELEASE_DIR/"

# 创建部署用的配置文件模板
cat > "$RELEASE_DIR/config.yaml.template" << EOF
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "127.0.0.1"
  port: 3306
  username: "root"
  password: "your_password"
  database: "orchestrator"
  charset: "utf8mb4"

logging:
  level: "info"
  format: "json"
EOF

# 创建启动脚本
cat > "$RELEASE_DIR/start.sh" << 'EOF'
#!/bin/bash
echo "启动工作流编排系统..."
./orchestrator-linux-amd64
EOF
chmod +x "$RELEASE_DIR/start.sh"

# 创建Dockerfile
cat > "$RELEASE_DIR/Dockerfile" << EOF
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY orchestrator-linux-amd64 ./orchestrator
COPY web ./web
COPY config.yaml.template ./config.yaml

EXPOSE 8080
CMD ["./orchestrator"]
EOF

# 生成构建信息文件
cat > "$RELEASE_DIR/build-info.json" << EOF
{
  "version": "$BUILD_VERSION",
  "build_time": "$BUILD_TIME",
  "build_commit": "$BUILD_COMMIT",
  "build_branch": "$BUILD_BRANCH",
  "go_version": "$(go version | awk '{print $3}')",
  "node_version": "$(node --version)"
}
EOF

# 创建压缩包
log_info "创建发布包..."
cd build
tar -czf "orchestrator-$BUILD_VERSION.tar.gz" "orchestrator-$BUILD_VERSION"
cd "$PROJECT_ROOT"

# 计算文件大小和哈希
RELEASE_SIZE=$(du -h "build/orchestrator-$BUILD_VERSION.tar.gz" | cut -f1)
RELEASE_HASH=$(sha256sum "build/orchestrator-$BUILD_VERSION.tar.gz" | cut -d' ' -f1)

echo ""
echo "==================== 构建完成 ===================="
log_success "版本: $BUILD_VERSION"
log_success "包大小: $RELEASE_SIZE"
log_success "SHA256: $RELEASE_HASH"
log_success "发布包: build/orchestrator-$BUILD_VERSION.tar.gz"
echo ""
echo "📁 构建产物:"
echo "   - 后端二进制: bin/"
echo "   - 前端静态文件: web/frontend/dist/"
echo "   - 发布包目录: build/orchestrator-$BUILD_VERSION/"
echo "   - 压缩包: build/orchestrator-$BUILD_VERSION.tar.gz"
echo ""
echo "🚀 部署说明:"
echo "   1. 解压发布包到目标服务器"
echo "   2. 复制 config.yaml.template 为 config.yaml 并修改配置"
echo "   3. 运行 ./start.sh 启动服务"
echo "=================================================="