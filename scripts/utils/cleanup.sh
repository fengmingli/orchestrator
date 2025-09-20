#!/bin/bash

# 项目清理脚本
# 作者: Orchestrator Team
# 版本: v1.0
# 描述: 清理项目中的临时文件、构建产物和缓存

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

# 计算目录大小
get_dir_size() {
    if [ -d "$1" ]; then
        du -sh "$1" 2>/dev/null | cut -f1
    else
        echo "0B"
    fi
}

# 清理类型
CLEANUP_TYPE=${1:-"all"}  # all, build, cache, logs, deps, temp

echo "=== 项目清理脚本 ==="

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

log_info "项目根目录: $PROJECT_ROOT"
log_info "清理类型: $CLEANUP_TYPE"

# 记录清理前的大小
TOTAL_FREED=0

# 清理构建产物
cleanup_build() {
    log_info "清理构建产物..."
    
    local freed=0
    
    # Go构建产物
    if [ -d "bin/" ]; then
        local size=$(get_dir_size "bin/")
        rm -rf bin/
        log_success "已清理 bin/ ($size)"
        freed=$((freed + $(echo $size | sed 's/[^0-9]//g' || echo 0)))
    fi
    
    # 前端构建产物
    if [ -d "web/frontend/dist/" ]; then
        local size=$(get_dir_size "web/frontend/dist/")
        rm -rf web/frontend/dist/
        log_success "已清理 web/frontend/dist/ ($size)"
    fi
    
    # 构建目录
    if [ -d "build/" ]; then
        local size=$(get_dir_size "build/")
        rm -rf build/
        log_success "已清理 build/ ($size)"
    fi
    
    # 清理可执行文件
    find . -name "orchestrator" -type f -not -path "./bin/*" -delete 2>/dev/null || true
    find . -name "*.exe" -type f -not -path "./web/frontend/node_modules/*" -delete 2>/dev/null || true
    
    log_success "构建产物清理完成"
}

# 清理缓存文件
cleanup_cache() {
    log_info "清理缓存文件..."
    
    # Go模块缓存 (谨慎清理，只清理项目相关)
    if [ -d "vendor/" ]; then
        local size=$(get_dir_size "vendor/")
        rm -rf vendor/
        log_success "已清理 vendor/ ($size)"
    fi
    
    # 前端缓存
    if [ -d "web/frontend/.vite/" ]; then
        local size=$(get_dir_size "web/frontend/.vite/")
        rm -rf web/frontend/.vite/
        log_success "已清理 web/frontend/.vite/ ($size)"
    fi
    
    if [ -d "web/frontend/.nuxt/" ]; then
        rm -rf web/frontend/.nuxt/
        log_success "已清理 web/frontend/.nuxt/"
    fi
    
    # 测试缓存
    if [ -d "test-reports/" ]; then
        local size=$(get_dir_size "test-reports/")
        rm -rf test-reports/
        log_success "已清理 test-reports/ ($size)"
    fi
    
    # Go测试缓存
    go clean -cache 2>/dev/null || true
    go clean -testcache 2>/dev/null || true
    
    log_success "缓存文件清理完成"
}

# 清理日志文件
cleanup_logs() {
    log_info "清理日志文件..."
    
    # 应用日志
    find . -name "*.log" -type f -not -path "./web/frontend/node_modules/*" -delete 2>/dev/null || true
    
    # 日志目录
    if [ -d "logs/" ]; then
        local size=$(get_dir_size "logs/")
        rm -rf logs/
        log_success "已清理 logs/ ($size)"
    fi
    
    # PID文件
    find . -name "*.pid" -type f -delete 2>/dev/null || true
    
    log_success "日志文件清理完成"
}

# 清理依赖文件
cleanup_deps() {
    log_info "清理依赖文件..."
    
    # Node.js依赖
    if [ -d "web/frontend/node_modules/" ]; then
        local size=$(get_dir_size "web/frontend/node_modules/")
        rm -rf web/frontend/node_modules/
        rm -f web/frontend/package-lock.json
        log_success "已清理 web/frontend/node_modules/ ($size)"
        log_warning "需要重新运行 npm install"
    fi
    
    # Go依赖 (谨慎操作)
    if [ "$FORCE_CLEAN_GO_MOD" = "true" ]; then
        go clean -modcache 2>/dev/null || true
        log_warning "已清理 Go 模块缓存"
    fi
    
    log_success "依赖文件清理完成"
}

# 清理临时文件
cleanup_temp() {
    log_info "清理临时文件..."
    
    # 临时目录
    for dir in tmp temp .tmp; do
        if [ -d "$dir/" ]; then
            local size=$(get_dir_size "$dir/")
            rm -rf "$dir/"
            log_success "已清理 $dir/ ($size)"
        fi
    done
    
    # 各种临时文件
    find . -name "*.tmp" -type f -delete 2>/dev/null || true
    find . -name "*.temp" -type f -delete 2>/dev/null || true
    find . -name ".DS_Store" -type f -delete 2>/dev/null || true
    find . -name "Thumbs.db" -type f -delete 2>/dev/null || true
    find . -name "*.swp" -type f -delete 2>/dev/null || true
    find . -name "*.swo" -type f -delete 2>/dev/null || true
    find . -name "*~" -type f -delete 2>/dev/null || true
    
    # 编辑器临时文件
    find . -name ".vscode" -type d -not -path "./web/frontend/node_modules/*" -exec rm -rf {} + 2>/dev/null || true
    find . -name ".idea" -type d -not -path "./web/frontend/node_modules/*" -exec rm -rf {} + 2>/dev/null || true
    
    # 覆盖率文件
    find . -name "coverage.out" -type f -delete 2>/dev/null || true
    find . -name "coverage" -type d -not -path "./web/frontend/node_modules/*" -exec rm -rf {} + 2>/dev/null || true
    
    log_success "临时文件清理完成"
}

# 清理Docker相关
cleanup_docker() {
    if ! command -v docker &> /dev/null; then
        return 0
    fi
    
    log_info "清理Docker相关..."
    
    # 停止并删除测试容器
    docker ps -a --filter "name=orchestrator-test" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
    
    # 清理未使用的镜像
    if [ "$FORCE_CLEAN_DOCKER" = "true" ]; then
        docker system prune -f 2>/dev/null || true
        log_success "Docker 系统清理完成"
    fi
}

# 显示清理统计
show_cleanup_stats() {
    echo ""
    echo "==================== 清理统计 ===================="
    
    # 显示当前目录大小
    CURRENT_SIZE=$(du -sh . 2>/dev/null | cut -f1)
    log_info "当前项目大小: $CURRENT_SIZE"
    
    # 检查剩余的大文件/目录
    log_info "大文件/目录 (>10MB):"
    find . -type f -size +10M -not -path "./web/frontend/node_modules/*" -not -path "./.git/*" 2>/dev/null | head -10 | while read file; do
        size=$(du -sh "$file" 2>/dev/null | cut -f1)
        echo "  - $file ($size)"
    done
    
    find . -type d -not -path "./web/frontend/node_modules/*" -not -path "./.git/*" 2>/dev/null | while read dir; do
        if [ -d "$dir" ]; then
            size=$(du -sh "$dir" 2>/dev/null | cut -f1 | sed 's/[^0-9.]//g')
            if [ -n "$size" ] && (( $(echo "$size > 10" | bc -l 2>/dev/null || echo 0) )); then
                echo "  - $dir ($(du -sh "$dir" 2>/dev/null | cut -f1))"
            fi
        fi
    done | head -5
    
    echo "=================================================="
}

# 安全确认
confirm_cleanup() {
    if [ "$CLEANUP_TYPE" = "all" ] || [ "$CLEANUP_TYPE" = "deps" ]; then
        echo ""
        log_warning "注意: 清理依赖文件后需要重新安装"
        read -p "确认继续? [y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "取消清理"
            exit 0
        fi
    fi
}

# 主函数
main() {
    # 显示清理前的状态
    BEFORE_SIZE=$(du -sh . 2>/dev/null | cut -f1)
    log_info "清理前项目大小: $BEFORE_SIZE"
    
    # 安全确认
    if [ "$CI" != "true" ]; then
        confirm_cleanup
    fi
    
    case "$CLEANUP_TYPE" in
        "build")
            cleanup_build
            ;;
        "cache")
            cleanup_cache
            ;;
        "logs")
            cleanup_logs
            ;;
        "deps")
            cleanup_deps
            ;;
        "temp")
            cleanup_temp
            ;;
        "docker")
            cleanup_docker
            ;;
        "all")
            cleanup_build
            cleanup_cache
            cleanup_logs
            cleanup_temp
            cleanup_docker
            ;;
        *)
            log_error "未知的清理类型: $CLEANUP_TYPE"
            echo "支持的清理类型: build, cache, logs, deps, temp, docker, all"
            exit 1
            ;;
    esac
    
    # 显示清理后的状态
    show_cleanup_stats
    
    AFTER_SIZE=$(du -sh . 2>/dev/null | cut -f1)
    echo ""
    log_success "🧹 清理完成!"
    log_info "清理前: $BEFORE_SIZE → 清理后: $AFTER_SIZE"
    
    # 给出后续建议
    echo ""
    echo "💡 后续操作建议:"
    if [ "$CLEANUP_TYPE" = "all" ] || [ "$CLEANUP_TYPE" = "deps" ]; then
        echo "   - 运行 'npm install' 重新安装前端依赖"
    fi
    if [ "$CLEANUP_TYPE" = "all" ] || [ "$CLEANUP_TYPE" = "build" ]; then
        echo "   - 运行 'scripts/build/build.sh' 重新构建项目"
    fi
    echo "   - 运行 'scripts/dev/start.sh' 启动开发环境"
}

# 运行主函数
main