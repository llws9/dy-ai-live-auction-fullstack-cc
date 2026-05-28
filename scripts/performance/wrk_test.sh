#!/bin/bash

# 使用wrk进行简单性能测试
# 用法: ./wrk_test.sh [base_url] [duration] [threads] [connections]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 默认配置
BASE_URL=${1:-"http://localhost:8080"}
DURATION=${2:-"30s"}
THREADS=${3:-4}
CONNECTIONS=${4:-100}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="${SCRIPT_DIR}/reports"

# 创建报告目录
mkdir -p "${REPORT_DIR}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}wrk 性能测试${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查wrk是否安装
if ! command -v wrk &> /dev/null; then
    echo -e "${RED}错误: wrk 未安装${NC}"
    echo ""
    echo -e "${YELLOW}安装方法:${NC}"
    echo -e "${YELLOW}macOS: brew install wrk${NC}"
    echo -e "${YELLOW}Ubuntu: sudo apt-get install wrk${NC}"
    exit 1
fi

echo -e "${GREEN}✓ wrk 已安装${NC}"

# 测试配置
echo -e "${GREEN}测试配置:${NC}"
echo -e "  目标地址: ${BASE_URL}"
echo -e "  持续时间: ${DURATION}"
echo -e "  线程数: ${THREADS}"
echo -e "  连接数: ${CONNECTIONS}"
echo ""

# 定义测试场景
declare -A TEST_SCENARIOS=(
    ["健康检查"]="/health"
    ["API根路径"]="/api/v1"
    ["竞拍列表"]="/api/v1/auctions?page=1&limit=20"
    ["商品列表"]="/api/v1/products?page=1&limit=20"
)

# 运行单个测试
run_wrk_test() {
    local test_name=$1
    local endpoint=$2

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}测试: ${test_name}${NC}"
    echo -e "${GREEN}端点: ${endpoint}${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""

    # 运行wrk测试
    wrk -t${THREADS} -c${CONNECTIONS} -d${DURATION} \
        --latency \
        "${BASE_URL}${endpoint}" | tee "${REPORT_DIR}/wrk_${test_name// /_}_$(date +%Y%m%d_%H%M%S).txt"

    echo ""
    echo -e "${GREEN}✓ ${test_name} 完成${NC}"
    echo ""
}

# 主菜单
echo -e "${YELLOW}选择测试场景:${NC}"
echo "1) 运行所有测试"
echo "2) 健康检查"
echo "3) API根路径"
echo "4) 竞拍列表"
echo "5) 商品列表"
echo "6) 自定义端点"
echo "0) 退出"
echo ""

read -p "请选择 (0-6): " choice

case $choice in
    1)
        echo -e "${YELLOW}运行所有测试...${NC}"
        echo ""
        for test_name in "${!TEST_SCENARIOS[@]}"; do
            run_wrk_test "${test_name}" "${TEST_SCENARIOS[$test_name]}"
        done
        ;;
    2)
        run_wrk_test "健康检查" "/health"
        ;;
    3)
        run_wrk_test "API根路径" "/api/v1"
        ;;
    4)
        run_wrk_test "竞拍列表" "/api/v1/auctions?page=1&limit=20"
        ;;
    5)
        run_wrk_test "商品列表" "/api/v1/products?page=1&limit=20"
        ;;
    6)
        read -p "输入自定义端点 (例如: /api/v1/auctions/1): " custom_endpoint
        run_wrk_test "自定义端点" "${custom_endpoint}"
        ;;
    0)
        echo -e "${YELLOW}退出${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}无效选择${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}测试完成!${NC}"
echo -e "${GREEN}报告保存在: ${REPORT_DIR}${NC}"
