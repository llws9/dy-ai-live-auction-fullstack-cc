#!/bin/bash

# 性能测试运行脚本
# 用法: ./run_performance_test.sh [test_type] [base_url]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
TEST_TYPE=${1:-"load"}
BASE_URL=${2:-"http://localhost:8080"}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="${SCRIPT_DIR}/reports"

# 创建报告目录
mkdir -p "${REPORT_DIR}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}性能测试运行脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查k6是否安装
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}错误: k6 未安装${NC}"
    echo -e "${YELLOW}请访问 https://k6.io/docs/getting-started/installation/ 安装k6${NC}"
    exit 1
fi

echo -e "${GREEN}✓ k6 已安装: $(k6 version)${NC}"

# 检查后端服务是否运行
echo -e "${YELLOW}检查后端服务...${NC}"
if curl -s "${BASE_URL}/health" > /dev/null; then
    echo -e "${GREEN}✓ 后端服务运行正常: ${BASE_URL}${NC}"
else
    echo -e "${YELLOW}⚠ 无法连接到后端服务: ${BASE_URL}${NC}"
    echo -e "${YELLOW}请确保后端服务已启动${NC}"
    read -p "是否继续测试? (y/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}测试配置:${NC}"
echo -e "  测试类型: ${TEST_TYPE}"
echo -e "  目标地址: ${BASE_URL}"
echo -e "  报告目录: ${REPORT_DIR}"
echo ""

# 设置环境变量
export BASE_URL="${BASE_URL}"

# 运行测试
run_test() {
    local test_name=$1
    local script_path=$2
    local report_prefix=$3

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}运行 ${test_name}...${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""

    # 运行k6测试
    k6 run \
        --out json="${REPORT_DIR}/${report_prefix}_$(date +%Y%m%d_%H%M%S).json" \
        "${script_path}"

    echo ""
    echo -e "${GREEN}✓ ${test_name} 完成${NC}"
    echo -e "${GREEN}报告已保存到: ${REPORT_DIR}/${report_prefix}_*.json${NC}"
    echo ""
}

# 根据测试类型运行不同的测试
case "${TEST_TYPE}" in
    "load")
        run_test "负载测试" "${SCRIPT_DIR}/load_test.js" "load_test"
        ;;
    "stress")
        run_test "压力测试" "${SCRIPT_DIR}/stress_test.js" "stress_test"
        ;;
    "spike")
        run_test "峰值测试" "${SCRIPT_DIR}/spike_test.js" "spike_test"
        ;;
    "all")
        echo -e "${YELLOW}运行所有测试...${NC}"
        echo ""

        run_test "负载测试" "${SCRIPT_DIR}/load_test.js" "load_test"
        run_test "压力测试" "${SCRIPT_DIR}/stress_test.js" "stress_test"
        run_test "峰值测试" "${SCRIPT_DIR}/spike_test.js" "spike_test"

        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}所有测试已完成${NC}"
        echo -e "${GREEN}========================================${NC}"
        ;;
    "scenario:login")
        run_test "并发登录测试" "${SCRIPT_DIR}/scenarios/concurrent_login.js" "scenario_login"
        ;;
    "scenario:register")
        run_test "并发注册测试" "${SCRIPT_DIR}/scenarios/concurrent_registration.js" "scenario_register"
        ;;
    "scenario:bid")
        run_test "并发出价测试" "${SCRIPT_DIR}/scenarios/concurrent_bid.js" "scenario_bid"
        ;;
    "scenario:query")
        run_test "竞拍查询测试" "${SCRIPT_DIR}/scenarios/auction_query.js" "scenario_query"
        ;;
    "scenario:websocket")
        run_test "WebSocket测试" "${SCRIPT_DIR}/scenarios/websocket_test.js" "scenario_websocket"
        ;;
    "scenario:product")
        run_test "商品管理测试" "${SCRIPT_DIR}/scenarios/product_management.js" "scenario_product"
        ;;
    "scenarios")
        echo -e "${YELLOW}运行所有测试场景...${NC}"
        echo ""

        run_test "并发登录测试" "${SCRIPT_DIR}/scenarios/concurrent_login.js" "scenario_login"
        run_test "并发注册测试" "${SCRIPT_DIR}/scenarios/concurrent_registration.js" "scenario_register"
        run_test "并发出价测试" "${SCRIPT_DIR}/scenarios/concurrent_bid.js" "scenario_bid"
        run_test "竞拍查询测试" "${SCRIPT_DIR}/scenarios/auction_query.js" "scenario_query"
        run_test "WebSocket测试" "${SCRIPT_DIR}/scenarios/websocket_test.js" "scenario_websocket"
        run_test "商品管理测试" "${SCRIPT_DIR}/scenarios/product_management.js" "scenario_product"

        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}所有测试场景已完成${NC}"
        echo -e "${GREEN}========================================${NC}"
        ;;
    *)
        echo -e "${RED}错误: 未知的测试类型 '${TEST_TYPE}'${NC}"
        echo ""
        echo -e "${YELLOW}可用的测试类型:${NC}"
        echo "  load              - 负载测试"
        echo "  stress            - 压力测试"
        echo "  spike             - 峰值测试"
        echo "  all               - 运行所有主要测试"
        echo ""
        echo "  scenario:login    - 并发登录测试"
        echo "  scenario:register - 并发注册测试"
        echo "  scenario:bid      - 并发出价测试"
        echo "  scenario:query    - 竞拍查询测试"
        echo "  scenario:websocket - WebSocket测试"
        echo "  scenario:product  - 商品管理测试"
        echo "  scenarios         - 运行所有测试场景"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}测试完成!${NC}"
echo -e "${GREEN}查看报告: open ${REPORT_DIR}${NC}"
