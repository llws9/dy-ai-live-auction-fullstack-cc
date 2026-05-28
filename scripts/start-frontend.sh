#!/bin/bash

# 前端服务启动脚本
# 用途: 正确启动H5用户端和Admin管理后台

set -e

echo "🚀 启动前端服务..."

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查端口是否被占用
check_port() {
    local port=$1
    if lsof -i :$port > /dev/null 2>&1; then
        echo "⚠️  端口 $port 已被占用"
        echo "尝试停止占用进程..."
        lsof -ti :$port | xargs kill -9 2>/dev/null || true
        sleep 1
    fi
}

# 检查并停止旧进程
echo "🔍 检查端口占用情况..."
check_port 5173
check_port 5175

# 启动H5用户端
echo -e "${BLUE}📱 启动H5用户端 (端口5173)...${NC}"
cd "$PROJECT_ROOT/frontend/h5"
npm run dev > /tmp/h5-auction.log 2>&1 &
H5_PID=$!
echo "H5用户端 PID: $H5_PID"

# 等待H5启动
sleep 3

# 验证H5是否启动成功
if curl -s http://localhost:5173 | grep -q "直播竞拍"; then
    echo -e "${GREEN}✅ H5用户端启动成功${NC}"
else
    echo "❌ H5用户端启动失败，请查看日志: /tmp/h5-auction.log"
    exit 1
fi

# 启动Admin后台
echo -e "${BLUE}💼 启动Admin管理后台 (端口5175)...${NC}"
cd "$PROJECT_ROOT/frontend/admin"
npm run dev > /tmp/admin-auction.log 2>&1 &
ADMIN_PID=$!
echo "Admin后台 PID: $ADMIN_PID"

# 等待Admin启动
sleep 3

# 验证Admin是否启动成功
if curl -s http://localhost:5175 | grep -q "竞拍管理后台"; then
    echo -e "${GREEN}✅ Admin管理后台启动成功${NC}"
else
    echo "❌ Admin管理后台启动失败，请查看日志: /tmp/admin-auction.log"
    exit 1
fi

echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}✅ 所有前端服务启动完成!${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""
echo "📱 H5用户端: http://localhost:5173"
echo "💼 Admin后台: http://localhost:5175"
echo ""
echo "📝 查看日志:"
echo "   H5:    tail -f /tmp/h5-auction.log"
echo "   Admin: tail -f /tmp/admin-auction.log"
echo ""
echo "🛑 停止服务:"
echo "   kill $H5_PID $ADMIN_PID"
