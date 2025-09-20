#!/bin/bash

# 工作流编排系统停止脚本 (兼容性脚本)
# 重定向到新的脚本位置

echo "⚠️  注意: 根目录的 stop.sh 已废弃，请使用 scripts/dev/stop.sh"
echo "🔄 自动重定向到新脚本..."
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 执行新的停止脚本
exec "$SCRIPT_DIR/scripts/dev/stop.sh" "$@"