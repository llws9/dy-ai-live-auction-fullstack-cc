#!/bin/bash

# 性能分析工具 - 分析测试结果并生成报告
# 用法: ./analyze_results.sh [report_file]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="${SCRIPT_DIR}/reports"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}性能测试结果分析${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查jq是否安装
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}警告: jq 未安装,部分功能可能不可用${NC}"
    echo -e "${YELLOW}安装: brew install jq (macOS) 或 apt-get install jq (Linux)${NC}"
fi

# 分析JSON报告
analyze_json_report() {
    local report_file=$1

    if [[ ! -f "${report_file}" ]]; then
        echo -e "${RED}错误: 报告文件不存在: ${report_file}${NC}"
        return 1
    fi

    echo -e "${GREEN}分析报告: ${report_file}${NC}"
    echo ""

    # 提取关键指标
    if command -v jq &> /dev/null; then
        local total_requests=$(jq -r '.metrics.http_reqs.values.count // 0' "${report_file}")
        local avg_duration=$(jq -r '.metrics.http_req_duration.values.avg // 0' "${report_file}")
        local p50_duration=$(jq -r '.metrics.http_req_duration.values["p(50)"] // 0' "${report_file}")
        local p99_duration=$(jq -r '.metrics.http_req_duration.values["p(99)"] // 0' "${report_file}")
        local error_rate=$(jq -r '.metrics.errors.values.rate // 0' "${report_file}")
        local rps=$(jq -r '.metrics.http_reqs.values.rate // 0' "${report_file}")

        echo -e "${BLUE}性能指标:${NC}"
        echo -e "  总请求数: ${total_requests}"
        echo -e "  平均响应时间: ${avg_duration}ms"
        echo -e "  P50响应时间: ${p50_duration}ms"
        echo -e "  P99响应时间: ${p99_duration}ms"
        echo -e "  错误率: $(echo "${error_rate} * 100" | bc -l | cut -c1-5)%"
        echo -e "  请求率: ${rps} req/s"
        echo ""

        # 性能评估
        echo -e "${BLUE}性能评估:${NC}"

        # P50评估
        if (( $(echo "${p50_duration} < 100" | bc -l) )); then
            echo -e "  ${GREEN}✓${NC} P50响应时间 < 100ms"
        else
            echo -e "  ${RED}✗${NC} P50响应时间 >= 100ms (目标: < 100ms)"
        fi

        # P99评估
        if (( $(echo "${p99_duration} < 200" | bc -l) )); then
            echo -e "  ${GREEN}✓${NC} P99响应时间 < 200ms"
        else
            echo -e "  ${RED}✗${NC} P99响应时间 >= 200ms (目标: < 200ms)"
        fi

        # 错误率评估
        if (( $(echo "${error_rate} < 0.01" | bc -l) )); then
            echo -e "  ${GREEN}✓${NC} 错误率 < 1%"
        else
            echo -e "  ${RED}✗${NC} 错误率 >= 1% (目标: < 1%)"
        fi

        # 吞吐量评估
        if (( $(echo "${rps} > 100" | bc -l) )); then
            echo -e "  ${GREEN}✓${NC} 吞吐量 > 100 req/s"
        else
            echo -e "  ${YELLOW}⚠${NC} 吞吐量 <= 100 req/s"
        fi
    else
        echo -e "${YELLOW}jq 未安装,无法解析JSON报告${NC}"
    fi

    echo ""
}

# 分析wrk文本报告
analyze_wrk_report() {
    local report_file=$1

    if [[ ! -f "${report_file}" ]]; then
        echo -e "${RED}错误: 报告文件不存在: ${report_file}${NC}"
        return 1
    fi

    echo -e "${GREEN}分析wrk报告: ${report_file}${NC}"
    echo ""

    # 提取关键信息
    local latency=$(grep "Latency" "${report_file}" | head -1)
    local req_per_sec=$(grep "Requests/sec" "${report_file}" | head -1)
    local transfer_per_sec=$(grep "Transfer/sec" "${report_file}" | head -1)

    echo -e "${BLUE}性能指标:${NC}"
    echo -e "  ${latency}"
    echo -e "  ${req_per_sec}"
    echo -e "  ${transfer_per_sec}"
    echo ""
}

# 主逻辑
if [[ $# -eq 0 ]]; then
    # 没有指定报告文件,列出所有可用的报告
    echo -e "${YELLOW}可用的测试报告:${NC}"
    echo ""

    echo -e "${GREEN}JSON报告:${NC}"
    ls -lht "${REPORT_DIR}"/*.json 2>/dev/null || echo "  无JSON报告"
    echo ""

    echo -e "${GREEN}HTML报告:${NC}"
    ls -lht "${REPORT_DIR}"/*.html 2>/dev/null || echo "  无HTML报告"
    echo ""

    echo -e "${GREEN}wrk报告:${NC}"
    ls -lht "${REPORT_DIR}"/*.txt 2>/dev/null || echo "  无wrk报告"
    echo ""

    echo -e "${YELLOW}用法: $0 <report_file>${NC}"
    echo -e "${YELLOW}示例: $0 reports/load_test_summary.json${NC}"
else
    REPORT_FILE=$1

    if [[ "${REPORT_FILE}" == *.json ]]; then
        analyze_json_report "${REPORT_FILE}"
    elif [[ "${REPORT_FILE}" == *.txt ]]; then
        analyze_wrk_report "${REPORT_FILE}"
    else
        echo -e "${RED}错误: 不支持的报告格式${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}分析完成${NC}"
echo -e "${GREEN}========================================${NC}"
