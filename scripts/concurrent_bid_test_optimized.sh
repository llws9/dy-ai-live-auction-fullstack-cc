#!/bin/bash

# 并发出价压力测试（优化版）
# 支持自动创建用户和真实并发测试

AUCTION_ID=4
BASE_URL="http://localhost:8082"
REQUESTS=100
START_USER_ID=1000

echo "====================================="
echo "   并发出价压力测试（优化版）"
echo "====================================="
echo ""
echo "竞拍ID: $AUCTION_ID"
echo "请求数: $REQUESTS"
echo "起始用户ID: $START_USER_ID"
echo ""

# 1. 批量创建测试用户
echo "【步骤1】批量创建测试用户..."
BATCH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/users/batch" \
  -H "Content-Type: application/json" \
  -d "{\"start_id\": $START_USER_ID, \"count\": $REQUESTS}")

echo "创建结果: $BATCH_RESPONSE"
echo ""

# 等待1秒确保用户创建完成
sleep 1

# 创建临时目录存储响应
TEMP_DIR="/tmp/bid_test_$$"
mkdir -p "$TEMP_DIR"

echo "【步骤2】开始并发出价测试..."
START_TIME=$(python3 -c 'import time; print(time.time() * 1000)')

# 并发送出价请求
for i in $(seq 1 $REQUESTS); do
  AMOUNT=$((100 + i))
  USER_ID=$((START_USER_ID + i - 1))

  curl -s -X POST "$BASE_URL/api/v1/auctions/$AUCTION_ID/bids" \
    -H "Content-Type: application/json" \
    -H "X-User-ID: $USER_ID" \
    -d "{\"amount\": $AMOUNT}" \
    > "$TEMP_DIR/response_$i.txt" &

  # 每10个请求输出一次进度
  if [ $((i % 10)) -eq 0 ]; then
    echo "已发送 $i 个请求..."
  fi
done

# 等待所有请求完成
wait

END_TIME=$(python3 -c 'import time; print(time.time() * 1000)')

echo ""
echo "【步骤3】统计结果"

# 统计成功和失败
SUCCESS=$(grep -l "\"success\":true" "$TEMP_DIR"/*.txt 2>/dev/null | wc -l | tr -d ' ')
FAIL=$(grep -l "\"success\":false" "$TEMP_DIR"/*.txt 2>/dev/null | wc -l | tr -d ' ')
ERROR=$(grep -l "\"code\":500" "$TEMP_DIR"/*.txt 2>/dev/null | wc -l | tr -d ' ')

echo "总请求数: $REQUESTS"
echo "成功: $SUCCESS"
echo "失败: $FAIL"
echo "服务器错误: $ERROR"
echo "成功率: $(python3 -c "print(round($SUCCESS / $REQUESTS * 100, 2))")%"
echo ""

# 计算QPS
DURATION=$(python3 -c "print(round(($END_TIME - $START_TIME) / 1000, 2))")
QPS=$(python3 -c "print(round($REQUESTS / $DURATION, 2))")
echo "总耗时: ${DURATION}s"
echo "QPS: $QPS"
echo ""

# 显示失败原因分析
if [ "$FAIL" -gt 0 ]; then
  echo "【失败原因分析】"
  grep "\"success\":false" "$TEMP_DIR"/*.txt 2>/dev/null | \
    python3 -c "
import sys
import json
messages = {}
for line in sys.stdin:
    try:
        data = json.loads(line.split(':', 1)[1].strip())
        msg = data.get('message', 'Unknown')
        messages[msg] = messages.get(msg, 0) + 1
    except:
        pass
for msg, count in sorted(messages.items(), key=lambda x: x[1], reverse=True)[:5]:
    print(f'  {msg}: {count}次')
"
  echo ""
fi

# 显示最新的竞拍价格
echo "【最终竞拍状态】"
curl -s "$BASE_URL/api/v1/auctions/$AUCTION_ID" | python3 -m json.tool
echo ""

# 显示前5名排名
echo "【出价排名 TOP 5】"
curl -s "$BASE_URL/api/v1/auctions/$AUCTION_ID/ranking?limit=5" | python3 -m json.tool
echo ""

# 清理临时文件
rm -rf "$TEMP_DIR"

echo "====================================="
echo "   测试完成"
echo "====================================="
