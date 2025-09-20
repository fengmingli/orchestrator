#!/bin/bash

# 工作流编排系统开发环境启动脚本
# 作者: Orchestrator Team
# 版本: v1.0
# 描述: 一键启动开发环境，包含前后端服务

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

# 检查端口是否被占用
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        log_warning "端口 $port 已被占用，正在尝试释放..."
        lsof -ti:$port | xargs kill -9 2>/dev/null || true
        sleep 2
    fi
}

# 等待服务启动
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1
    
    log_info "等待 $service_name 启动..."
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            log_success "$service_name 启动成功"
            return 0
        fi
        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    log_error "$service_name 启动超时"
    return 1
}

echo "=== 工作流编排系统开发环境启动脚本 ==="

# 环境检查
log_info "检查运行环境..."
check_command "go" "Go 1.21+"
check_command "node" "Node.js 18+"
check_command "npm" "npm"

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

log_info "项目根目录: $PROJECT_ROOT"

# 检查端口占用
check_port 8080
check_port 3000

# 编译后端服务
log_info "编译后端服务..."
cd cmd/web
go build -o ../../bin/web main.go
if [ $? -ne 0 ]; then
    log_error "后端编译失败"
    exit 1
fi
cd "$PROJECT_ROOT"
log_success "后端编译完成"

# 处理前端依赖
log_info "检查前端依赖..."
cd web/frontend

if [ ! -d "node_modules" ] || [ ! -f "node_modules/.install-complete" ]; then
    log_info "安装前端依赖..."
    npm install
    if [ $? -ne 0 ]; then
        log_error "前端依赖安装失败"
        exit 1
    fi
    touch node_modules/.install-complete
    log_success "前端依赖安装完成"
else
    log_info "前端依赖已存在，跳过安装"
fi

cd "$PROJECT_ROOT"

# 启动后端服务
log_info "启动后端服务..."
# 创建日志目录
mkdir -p logs
nohup ./bin/web > logs/backend.log 2>&1 &
WEB_PID=$!
echo $WEB_PID > logs/backend.pid

# 等待后端启动
if wait_for_service "http://localhost:8080/health" "后端服务"; then
    log_success "后端服务启动成功 (PID: $WEB_PID)"
    log_info "API地址: http://localhost:8080"
    log_info "健康检查: http://localhost:8080/health"
    log_info "日志文件: logs/backend.log"
else
    log_error "后端服务启动失败，请检查日志 logs/backend.log"
    cat logs/backend.log | tail -20
    exit 1
fi

# 启动前端开发服务器
log_info "启动前端开发服务器..."
cd web/frontend

# 创建前端启动脚本
cat > start_frontend.sh << 'EOF'
#!/bin/bash
npm run dev
EOF
chmod +x start_frontend.sh

log_success "🚀 所有服务启动完成!"
echo ""
echo "==================== 服务信息 ===================="
echo "🌐 前端管理界面: http://localhost:3000"
echo "🔧 后端API接口:  http://localhost:8080"
echo "📊 健康检查:     http://localhost:8080/health"
echo "📝 后端日志:     logs/backend.log"
echo ""
echo "💡 使用说明:"
echo "   - 前端会自动打开浏览器窗口"
echo "   - 按 Ctrl+C 停止前端服务"
echo "   - 使用 scripts/dev/stop.sh 停止所有服务"
echo "=================================================="
echo ""

log_info "启动前端开发服务器..."
npm run dev