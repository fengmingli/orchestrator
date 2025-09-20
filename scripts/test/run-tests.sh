#!/bin/bash

# 自动化测试脚本
# 作者: Orchestrator Team
# 版本: v1.0
# 描述: 运行完整的测试套件，包括单元测试、集成测试和E2E测试

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

# 测试配置
TEST_TYPE=${1:-"all"}  # all, unit, integration, e2e
COVERAGE_THRESHOLD=80
GENERATE_REPORTS=${GENERATE_REPORTS:-"true"}
CI_MODE=${CI:-"false"}

echo "=== 工作流编排系统测试脚本 ==="

# 环境检查
log_info "检查测试环境..."
check_command "go" "Go 1.21+"
check_command "node" "Node.js 18+"
check_command "npm" "npm"

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

log_info "项目根目录: $PROJECT_ROOT"
log_info "测试类型: $TEST_TYPE"
log_info "覆盖率阈值: $COVERAGE_THRESHOLD%"

# 创建测试报告目录
TEST_REPORTS_DIR="test-reports"
mkdir -p "$TEST_REPORTS_DIR"

# 测试开始时间
TEST_START_TIME=$(date +%s)

# 后端单元测试
run_backend_unit_tests() {
    log_info "运行后端单元测试..."
    
    # 设置测试环境变量
    export GO_ENV=test
    export DB_HOST=localhost
    export DB_PORT=3307  # 使用不同端口避免冲突
    export DB_USERNAME=test
    export DB_PASSWORD=test
    export DB_DATABASE=orchestrator_test
    
    # 运行测试并生成覆盖率报告
    go test -v -race -coverprofile="$TEST_REPORTS_DIR/backend-coverage.out" ./...
    
    if [ $? -ne 0 ]; then
        log_error "后端单元测试失败"
        return 1
    fi
    
    # 生成覆盖率报告
    if [ "$GENERATE_REPORTS" = "true" ]; then
        go tool cover -html="$TEST_REPORTS_DIR/backend-coverage.out" -o "$TEST_REPORTS_DIR/backend-coverage.html"
        
        # 计算覆盖率
        BACKEND_COVERAGE=$(go tool cover -func="$TEST_REPORTS_DIR/backend-coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
        
        if (( $(echo "$BACKEND_COVERAGE < $COVERAGE_THRESHOLD" | bc -l) )); then
            log_warning "后端代码覆盖率 ($BACKEND_COVERAGE%) 低于阈值 ($COVERAGE_THRESHOLD%)"
        else
            log_success "后端代码覆盖率: $BACKEND_COVERAGE%"
        fi
    fi
    
    log_success "后端单元测试完成"
    return 0
}

# 前端单元测试
run_frontend_unit_tests() {
    log_info "运行前端单元测试..."
    
    cd web/frontend
    
    # 安装依赖
    if [ ! -d "node_modules" ]; then
        npm ci
    fi
    
    # 运行测试
    if [ "$GENERATE_REPORTS" = "true" ]; then
        npm run test:coverage -- --reporter=junit --outputFile="../../$TEST_REPORTS_DIR/frontend-test-results.xml"
    else
        npm run test
    fi
    
    if [ $? -ne 0 ]; then
        log_error "前端单元测试失败"
        cd "$PROJECT_ROOT"
        return 1
    fi
    
    # 复制覆盖率报告
    if [ "$GENERATE_REPORTS" = "true" ] && [ -d "coverage" ]; then
        cp -r coverage "../../$TEST_REPORTS_DIR/frontend-coverage"
        
        # 提取覆盖率数据
        if [ -f "coverage/lcov-report/index.html" ]; then
            FRONTEND_COVERAGE=$(grep -o "Functions[^0-9]*[0-9]*\.[0-9]*%" "coverage/lcov-report/index.html" | head -1 | grep -o "[0-9]*\.[0-9]*" || echo "0")
            
            if (( $(echo "$FRONTEND_COVERAGE < $COVERAGE_THRESHOLD" | bc -l) )); then
                log_warning "前端代码覆盖率 ($FRONTEND_COVERAGE%) 低于阈值 ($COVERAGE_THRESHOLD%)"
            else
                log_success "前端代码覆盖率: $FRONTEND_COVERAGE%"
            fi
        fi
    fi
    
    cd "$PROJECT_ROOT"
    log_success "前端单元测试完成"
    return 0
}

# 集成测试
run_integration_tests() {
    log_info "运行集成测试..."
    
    # 启动测试数据库
    if command -v docker &> /dev/null; then
        log_info "启动测试数据库..."
        docker run -d --name orchestrator-test-db \
            -e MYSQL_ROOT_PASSWORD=test123 \
            -e MYSQL_DATABASE=orchestrator_test \
            -e MYSQL_USER=test \
            -e MYSQL_PASSWORD=test \
            -p 3307:3306 \
            mysql:8.0
        
        # 等待数据库启动
        sleep 30
        
        # 运行集成测试
        export GO_ENV=integration
        export DB_HOST=localhost
        export DB_PORT=3307
        export DB_USERNAME=test
        export DB_PASSWORD=test
        export DB_DATABASE=orchestrator_test
        
        go test -v -tags=integration ./test/integration/...
        
        # 清理测试数据库
        docker stop orchestrator-test-db && docker rm orchestrator-test-db
    else
        log_warning "Docker未安装，跳过集成测试"
        return 0
    fi
    
    if [ $? -ne 0 ]; then
        log_error "集成测试失败"
        return 1
    fi
    
    log_success "集成测试完成"
    return 0
}

# E2E测试
run_e2e_tests() {
    log_info "运行E2E测试..."
    
    # 启动应用
    log_info "启动测试环境..."
    ./scripts/dev/start.sh &
    START_PID=$!
    
    # 等待服务启动
    sleep 10
    
    # 检查服务是否正常
    if ! curl -s http://localhost:8080/health > /dev/null; then
        log_error "测试环境启动失败"
        kill $START_PID 2>/dev/null || true
        return 1
    fi
    
    cd web/frontend
    
    # 运行Cypress E2E测试
    if [ "$CI_MODE" = "true" ]; then
        npm run cypress:run
    else
        npm run cypress:run --headless
    fi
    
    E2E_RESULT=$?
    
    cd "$PROJECT_ROOT"
    
    # 停止测试环境
    ./scripts/dev/stop.sh
    
    if [ $E2E_RESULT -ne 0 ]; then
        log_error "E2E测试失败"
        return 1
    fi
    
    log_success "E2E测试完成"
    return 0
}

# 性能测试
run_performance_tests() {
    log_info "运行性能测试..."
    
    # 检查是否安装了wrk或ab
    if command -v wrk &> /dev/null; then
        PERF_TOOL="wrk"
    elif command -v ab &> /dev/null; then
        PERF_TOOL="ab"
    else
        log_warning "未安装性能测试工具 (wrk/ab)，跳过性能测试"
        return 0
    fi
    
    # 启动应用
    ./scripts/dev/start.sh &
    START_PID=$!
    sleep 10
    
    # 运行性能测试
    log_info "使用 $PERF_TOOL 进行性能测试..."
    
    if [ "$PERF_TOOL" = "wrk" ]; then
        wrk -t4 -c100 -d30s --latency http://localhost:8080/api/v1/steps > "$TEST_REPORTS_DIR/performance-report.txt"
    elif [ "$PERF_TOOL" = "ab" ]; then
        ab -n 1000 -c 10 http://localhost:8080/api/v1/steps > "$TEST_REPORTS_DIR/performance-report.txt"
    fi
    
    # 停止应用
    ./scripts/dev/stop.sh
    
    log_success "性能测试完成"
    return 0
}

# 生成测试报告
generate_test_report() {
    if [ "$GENERATE_REPORTS" != "true" ]; then
        return 0
    fi
    
    log_info "生成测试报告..."
    
    TEST_END_TIME=$(date +%s)
    TEST_DURATION=$((TEST_END_TIME - TEST_START_TIME))
    
    cat > "$TEST_REPORTS_DIR/test-summary.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>测试报告 - 工作流编排系统</title>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .success { color: green; }
        .warning { color: orange; }
        .error { color: red; }
        .info { color: blue; }
    </style>
</head>
<body>
    <div class="header">
        <h1>工作流编排系统 - 测试报告</h1>
        <p><strong>生成时间:</strong> $(date)</p>
        <p><strong>测试类型:</strong> $TEST_TYPE</p>
        <p><strong>测试耗时:</strong> ${TEST_DURATION}秒</p>
        <p><strong>项目版本:</strong> $(git describe --tags --always 2>/dev/null || echo "unknown")</p>
    </div>
    
    <div class="section">
        <h2>测试结果</h2>
        <ul>
            <li class="$([ -f "$TEST_REPORTS_DIR/backend-coverage.out" ] && echo "success" || echo "info")">
                后端单元测试: $([ -f "$TEST_REPORTS_DIR/backend-coverage.out" ] && echo "✅ 通过" || echo "⚪ 未运行")
            </li>
            <li class="$([ -d "$TEST_REPORTS_DIR/frontend-coverage" ] && echo "success" || echo "info")">
                前端单元测试: $([ -d "$TEST_REPORTS_DIR/frontend-coverage" ] && echo "✅ 通过" || echo "⚪ 未运行")
            </li>
            <li class="info">集成测试: ⚪ 需要Docker支持</li>
            <li class="info">E2E测试: ⚪ 需要完整环境</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>覆盖率报告</h2>
        <ul>
            $([ -f "$TEST_REPORTS_DIR/backend-coverage.html" ] && echo '<li><a href="backend-coverage.html">后端代码覆盖率</a></li>')
            $([ -d "$TEST_REPORTS_DIR/frontend-coverage" ] && echo '<li><a href="frontend-coverage/index.html">前端代码覆盖率</a></li>')
        </ul>
    </div>
    
    <div class="section">
        <h2>性能测试</h2>
        $([ -f "$TEST_REPORTS_DIR/performance-report.txt" ] && echo '<pre>' && cat "$TEST_REPORTS_DIR/performance-report.txt" && echo '</pre>' || echo '<p>未运行性能测试</p>')
    </div>
</body>
</html>
EOF
    
    log_success "测试报告已生成: $TEST_REPORTS_DIR/test-summary.html"
}

# 主函数
main() {
    case "$TEST_TYPE" in
        "unit")
            run_backend_unit_tests && run_frontend_unit_tests
            ;;
        "integration")
            run_integration_tests
            ;;
        "e2e")
            run_e2e_tests
            ;;
        "performance")
            run_performance_tests
            ;;
        "all")
            run_backend_unit_tests && \
            run_frontend_unit_tests && \
            run_integration_tests && \
            run_e2e_tests && \
            run_performance_tests
            ;;
        *)
            log_error "未知的测试类型: $TEST_TYPE"
            echo "支持的测试类型: unit, integration, e2e, performance, all"
            exit 1
            ;;
    esac
    
    TESTS_RESULT=$?
    
    # 生成报告
    generate_test_report
    
    echo ""
    echo "==================== 测试完成 ===================="
    if [ $TESTS_RESULT -eq 0 ]; then
        log_success "所有测试通过 ✅"
    else
        log_error "部分测试失败 ❌"
    fi
    
    log_info "测试报告: $TEST_REPORTS_DIR/"
    log_info "测试耗时: $(($(date +%s) - TEST_START_TIME))秒"
    echo "=================================================="
    
    exit $TESTS_RESULT
}

# 运行主函数
main