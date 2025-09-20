#!/bin/bash

# 系统健康检查脚本
# 作者: Orchestrator Team
# 版本: v1.0
# 描述: 检查系统各组件运行状态和健康度

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

# 检查HTTP服务
check_http_service() {
    local url=$1
    local service_name=$2
    local timeout=${3:-5}
    
    if curl -s --max-time $timeout "$url" > /dev/null 2>&1; then
        log_success "$service_name 运行正常"
        return 0
    else
        log_error "$service_name 无响应"
        return 1
    fi
}

# 检查端口占用
check_port() {
    local port=$1
    local service_name=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        local pid=$(lsof -Pi :$port -sTCP:LISTEN -t)
        local process=$(ps -p $pid -o comm= 2>/dev/null || echo "unknown")
        log_success "$service_name (端口 $port) 正在运行 (PID: $pid, 进程: $process)"
        return 0
    else
        log_warning "$service_name (端口 $port) 未运行"
        return 1
    fi
}

# 检查数据库连接
check_database() {
    local host=${DB_HOST:-"127.0.0.1"}
    local port=${DB_PORT:-"3306"}
    local username=${DB_USERNAME:-"root"}
    local password=${DB_PASSWORD:-"root123456"}
    local database=${DB_DATABASE:-"orchestrator"}
    
    log_info "检查数据库连接..."
    
    if command -v mysql &> /dev/null; then
        if mysql -h "$host" -P "$port" -u "$username" -p"$password" -e "USE $database; SELECT 1;" 2>/dev/null; then
            log_success "MySQL 数据库连接正常"
            
            # 检查表是否存在
            local tables=$(mysql -h "$host" -P "$port" -u "$username" -p"$password" -D "$database" -e "SHOW TABLES;" 2>/dev/null | tail -n +2 | wc -l)
            log_info "数据库表数量: $tables"
            
            return 0
        else
            log_warning "MySQL 数据库连接失败，可能使用 SQLite"
            
            # 检查SQLite数据库
            if [ -f "orchestrator.db" ]; then
                if command -v sqlite3 &> /dev/null; then
                    local tables=$(sqlite3 orchestrator.db ".tables" 2>/dev/null | wc -w)
                    log_success "SQLite 数据库文件存在，表数量: $tables"
                    return 0
                else
                    log_warning "SQLite 数据库文件存在，但未安装 sqlite3 命令"
                    return 1
                fi
            else
                log_error "未找到数据库文件"
                return 1
            fi
        fi
    else
        log_warning "未安装 mysql 客户端，跳过数据库检查"
        return 1
    fi
}

# 检查磁盘空间
check_disk_space() {
    log_info "检查磁盘空间..."
    
    local usage=$(df . | tail -1 | awk '{print $5}' | sed 's/%//')
    local available=$(df -h . | tail -1 | awk '{print $4}')
    
    if [ "$usage" -gt 90 ]; then
        log_error "磁盘空间不足 (使用率: ${usage}%, 可用: $available)"
        return 1
    elif [ "$usage" -gt 80 ]; then
        log_warning "磁盘空间紧张 (使用率: ${usage}%, 可用: $available)"
        return 1
    else
        log_success "磁盘空间充足 (使用率: ${usage}%, 可用: $available)"
        return 0
    fi
}

# 检查内存使用
check_memory() {
    log_info "检查内存使用..."
    
    if command -v free &> /dev/null; then
        local mem_info=$(free -h | grep "Mem:")
        local total=$(echo $mem_info | awk '{print $2}')
        local used=$(echo $mem_info | awk '{print $3}')
        local available=$(echo $mem_info | awk '{print $7}')
        
        local usage_percent=$(free | grep "Mem:" | awk '{printf "%.0f", $3/$2 * 100}')
        
        if [ "$usage_percent" -gt 90 ]; then
            log_error "内存使用率过高 (${usage_percent}%, 总量: $total, 已用: $used, 可用: $available)"
            return 1
        elif [ "$usage_percent" -gt 80 ]; then
            log_warning "内存使用率较高 (${usage_percent}%, 总量: $total, 已用: $used, 可用: $available)"
            return 1
        else
            log_success "内存使用正常 (${usage_percent}%, 总量: $total, 已用: $used, 可用: $available)"
            return 0
        fi
    else
        log_warning "无法获取内存信息"
        return 1
    fi
}

# 检查系统负载
check_load() {
    log_info "检查系统负载..."
    
    if [ -f "/proc/loadavg" ]; then
        local load=$(cat /proc/loadavg | awk '{print $1}')
        local cores=$(nproc 2>/dev/null || echo 1)
        local load_percent=$(echo "$load $cores" | awk '{printf "%.0f", $1/$2 * 100}')
        
        if [ "$load_percent" -gt 80 ]; then
            log_warning "系统负载较高 (${load_percent}%, 1分钟平均负载: $load, CPU核心数: $cores)"
            return 1
        else
            log_success "系统负载正常 (${load_percent}%, 1分钟平均负载: $load, CPU核心数: $cores)"
            return 0
        fi
    else
        log_warning "无法获取系统负载信息"
        return 1
    fi
}

# 检查日志文件
check_logs() {
    log_info "检查日志文件..."
    
    local error_count=0
    local log_files=("web.log" "logs/backend.log" "logs/frontend.log")
    
    for log_file in "${log_files[@]}"; do
        if [ -f "$log_file" ]; then
            local size=$(du -h "$log_file" | cut -f1)
            local errors=$(grep -c "ERROR\|FATAL\|PANIC" "$log_file" 2>/dev/null || echo 0)
            local warnings=$(grep -c "WARN" "$log_file" 2>/dev/null || echo 0)
            
            log_info "$log_file: 大小 $size, 错误 $errors 条, 警告 $warnings 条"
            
            if [ "$errors" -gt 10 ]; then
                log_warning "$log_file 包含较多错误 ($errors 条)"
                error_count=$((error_count + 1))
            fi
            
            # 显示最近的错误
            if [ "$errors" -gt 0 ]; then
                log_warning "最近的错误:"
                grep "ERROR\|FATAL\|PANIC" "$log_file" | tail -3 | while read line; do
                    echo "    $line"
                done
            fi
        fi
    done
    
    if [ "$error_count" -eq 0 ]; then
        log_success "日志文件检查正常"
        return 0
    else
        log_warning "发现 $error_count 个日志文件包含较多错误"
        return 1
    fi
}

# 检查API接口
check_api_endpoints() {
    log_info "检查API接口..."
    
    local base_url="http://localhost:8080"
    local endpoints=(
        "/health:健康检查"
        "/api/v1/steps:步骤管理"
        "/api/v1/templates:模板管理"
        "/api/v1/executions:执行管理"
    )
    
    local failed_count=0
    
    for endpoint_info in "${endpoints[@]}"; do
        local endpoint=$(echo "$endpoint_info" | cut -d: -f1)
        local name=$(echo "$endpoint_info" | cut -d: -f2)
        
        if curl -s --max-time 5 "$base_url$endpoint" > /dev/null 2>&1; then
            log_success "$name ($endpoint) 可访问"
        else
            log_error "$name ($endpoint) 无法访问"
            failed_count=$((failed_count + 1))
        fi
    done
    
    if [ "$failed_count" -eq 0 ]; then
        log_success "所有API接口检查通过"
        return 0
    else
        log_error "$failed_count 个API接口无法访问"
        return 1
    fi
}

# 生成健康报告
generate_health_report() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local report_file="health-report-$(date '+%Y%m%d-%H%M%S').json"
    
    cat > "$report_file" << EOF
{
  "timestamp": "$timestamp",
  "system": {
    "hostname": "$(hostname)",
    "uptime": "$(uptime -p 2>/dev/null || echo "unknown")",
    "os": "$(uname -s 2>/dev/null || echo "unknown")",
    "arch": "$(uname -m 2>/dev/null || echo "unknown")"
  },
  "services": {
    "backend": "$(check_port 8080 "后端服务" >/dev/null 2>&1 && echo "running" || echo "stopped")",
    "frontend": "$(check_port 3000 "前端服务" >/dev/null 2>&1 && echo "running" || echo "stopped")",
    "database": "$(check_database >/dev/null 2>&1 && echo "connected" || echo "disconnected")"
  },
  "resources": {
    "disk_usage": "$(df . | tail -1 | awk '{print $5}')",
    "memory_usage": "$(free | grep "Mem:" | awk '{printf "%.0f%%", $3/$2 * 100}' 2>/dev/null || echo "unknown")",
    "load_average": "$(cat /proc/loadavg 2>/dev/null | awk '{print $1}' || echo "unknown")"
  },
  "health_score": "$(echo "scale=2; (100 - $failed_checks * 10)" | bc -l 2>/dev/null || echo "unknown")"
}
EOF
    
    log_info "健康报告已生成: $report_file"
}

echo "=== 工作流编排系统健康检查 ==="

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

log_info "项目根目录: $PROJECT_ROOT"
log_info "检查时间: $(date)"

echo ""
echo "==================== 服务状态检查 ===================="

failed_checks=0

# 检查后端服务
if ! check_port 8080 "后端服务"; then
    failed_checks=$((failed_checks + 1))
fi

# 检查前端服务
if ! check_port 3000 "前端开发服务"; then
    failed_checks=$((failed_checks + 1))
fi

# 检查API接口
if ! check_api_endpoints; then
    failed_checks=$((failed_checks + 1))
fi

echo ""
echo "==================== 数据库检查 ===================="

# 检查数据库
if ! check_database; then
    failed_checks=$((failed_checks + 1))
fi

echo ""
echo "==================== 系统资源检查 ===================="

# 检查磁盘空间
if ! check_disk_space; then
    failed_checks=$((failed_checks + 1))
fi

# 检查内存使用
if ! check_memory; then
    failed_checks=$((failed_checks + 1))
fi

# 检查系统负载
if ! check_load; then
    failed_checks=$((failed_checks + 1))
fi

echo ""
echo "==================== 日志检查 ===================="

# 检查日志文件
if ! check_logs; then
    failed_checks=$((failed_checks + 1))
fi

echo ""
echo "==================== 健康检查总结 ===================="

# 计算健康分数
health_score=$(echo "scale=2; (100 - $failed_checks * 10)" | bc -l 2>/dev/null || echo "unknown")

if [ "$failed_checks" -eq 0 ]; then
    log_success "🎉 系统运行完全正常！健康分数: 100"
elif [ "$failed_checks" -le 2 ]; then
    log_warning "⚠️  系统运行基本正常，有 $failed_checks 项检查失败。健康分数: $health_score"
else
    log_error "❌ 系统存在问题，有 $failed_checks 项检查失败。健康分数: $health_score"
fi

echo ""
echo "💡 建议操作:"
if [ "$failed_checks" -gt 0 ]; then
    echo "   - 检查服务是否正常启动: scripts/dev/start.sh"
    echo "   - 查看详细日志: tail -f web.log"
    echo "   - 重启服务: scripts/dev/stop.sh && scripts/dev/start.sh"
fi
echo "   - 清理临时文件: scripts/utils/cleanup.sh temp"
echo "   - 查看更多监控: scripts/utils/monitor.sh"

# 生成健康报告
if [ "$GENERATE_REPORT" = "true" ]; then
    generate_health_report
fi

echo "=================================================="

exit $failed_checks