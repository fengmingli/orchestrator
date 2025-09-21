#!/bin/bash

# 工作流编排系统启动脚本
# 重定向到开发环境脚本

echo "🚀 启动工作流编排系统..."
echo "📍 使用开发环境配置"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 执行开发环境启动脚本
exec "$SCRIPT_DIR/dev/start.sh" "$@"