#!/bin/bash

# 工作流编排系统启动脚本

echo "=== 工作流编排系统启动脚本 ==="

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "错误: Go 未安装，请先安装 Go 1.21+"
    exit 1
fi

# 检查Node.js是否安装
if ! command -v node &> /dev/null; then
    echo "错误: Node.js 未安装，请先安装 Node.js 18+"
    exit 1
fi

# 检查npm是否安装
if ! command -v npm &> /dev/null; then
    echo "错误: npm 未安装，请先安装 npm"
    exit 1
fi

echo "1. 编译后端服务..."
cd cmd/web && go build -o ../../bin/web main.go
if [ $? -ne 0 ]; then
    echo "后端编译失败"
    exit 1
fi
cd ../..

echo "2. 安装前端依赖..."
cd web/frontend
if [ ! -d "node_modules" ]; then
    npm install
    if [ $? -ne 0 ]; then
        echo "前端依赖安装失败"
        exit 1
    fi
fi

echo "3. 构建前端项目..."
npm run build
if [ $? -ne 0 ]; then
    echo "前端构建失败"
    exit 1
fi
cd ../..

echo "4. 启动后端服务..."
nohup ./bin/web > web.log 2>&1 &
WEB_PID=$!

# 等待后端启动
sleep 3

# 检查后端是否启动成功
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ 后端服务启动成功 (PID: $WEB_PID)"
    echo "   - API地址: http://localhost:8080"
    echo "   - 健康检查: http://localhost:8080/health"
    echo "   - 日志文件: web.log"
else
    echo "❌ 后端服务启动失败，请检查日志 web.log"
    exit 1
fi

echo "5. 启动前端开发服务器..."
cd web/frontend
echo "启动前端开发服务器，访问 http://localhost:3000 查看管理界面"
echo "按 Ctrl+C 停止服务"
npm run dev