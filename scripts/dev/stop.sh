#!/bin/bash

# 工作流编排系统开发环境停止脚本
# 作者: Orchestrator Team
# 版本: v1.0
# 描述: 停止所有开发环境服务

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

# 停止进程
stop_process() {
    local process_name=$1
    local display_name=$2
    
    log_info "停止 $display_name..."
    
    # 方法1: 使用pkill
    if pkill -f "$process_name" 2>/dev/null; then
        sleep 2
        log_success "$display_name 已停止"
        return 0
    fi
    
    # 方法2: 使用lsof查找端口并终止
    local pids
    case "$process_name" in
        "./bin/web")
            pids=$(lsof -ti:8080 2>/dev/null || true)
            ;;
        "vite"|"npm run dev")
            pids=$(lsof -ti:3000 2>/dev/null || true)
            ;;
    esac
    
    if [ -n "$pids" ]; then
        echo "$pids" | xargs kill -TERM 2>/dev/null || true
        sleep 3
        echo "$pids" | xargs kill -KILL 2>/dev/null || true
        log_success "$display_name 已强制停止"
        return 0
    fi
    
    log_warning "$display_name 未在运行"
    return 0
}

# 停止通过PID文件记录的进程
stop_process_by_pid() {
    local pid_file=$1
    local display_name=$2
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        log_info "停止 $display_name (PID: $pid)..."
        
        if kill -TERM "$pid" 2>/dev/null; then
            # 等待进程优雅退出
            for i in {1..10}; do
                if ! kill -0 "$pid" 2>/dev/null; then
                    break
                fi
                sleep 1
            done
            
            # 如果进程仍然存在，强制终止
            if kill -0 "$pid" 2>/dev/null; then
                kill -KILL "$pid" 2>/dev/null || true
            fi
            
            rm -f "$pid_file"
            log_success "$display_name 已停止"
        else
            log_warning "$display_name 进程不存在，清理PID文件"
            rm -f "$pid_file"
        fi
    fi
}

echo "=== 工作流编排系统开发环境停止脚本 ==="

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

# 停止后端服务
stop_process_by_pid "logs/backend.pid" "后端服务"
stop_process "./bin/web" "后端服务"

# 停止前端服务
stop_process "vite" "前端开发服务器"
stop_process "npm run dev" "前端开发服务器"

# 清理可能残留的进程
log_info "清理残留进程..."
pkill -f "orchestrator" 2>/dev/null || true
pkill -f "bin/web" 2>/dev/null || true

# 显示最终状态
echo ""
echo "==================== 服务状态 ===================="

# 检查端口占用情况
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    log_warning "端口 8080 仍被占用"
    lsof -Pi :8080 -sTCP:LISTEN
else
    log_success "端口 8080 已释放"
fi

if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null 2>&1; then
    log_warning "端口 3000 仍被占用"
    lsof -Pi :3000 -sTCP:LISTEN
else
    log_success "端口 3000 已释放"
fi

echo "=================================================="
log_success "🛑 所有服务已停止"

# 清理日志文件 (可选)
read -p "是否清理日志文件? [y/N]: " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f logs/backend.log logs/backend.pid
    log_success "日志文件已清理"
fi