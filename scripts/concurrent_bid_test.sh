#!/bin/bash

# 并发出价压力测试
# 使用curl批量发送出价请求

AUCTION_ID=4
BASE_URL="http://localhost:8082"
REQUESTS=100

echo "====================================="
echo "   并发出价压力测试"
echo "====================================="
echo ""
echo "竞拍ID: $AUCTION_ID"
echo "请求数: $REQUESTS"
echo ""

# 创建临时目录存储响应
TEMP_DIR="/tmp/bid_test_$$"
mkdir -p "$TEMP_DIR"

echo "【开始测试】发送 $REQUESTS 个并发出价请求..."
START_TIME=$(python3 -c 'import time; print(time.time() * 1000)')

# 并发送出价请求
for i in $(seq 1 $REQUESTS); do
  AMOUNT=$((100 + i))
  USER_ID=$((1000 + i))

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
echo "【统计结果】"

# 统计成功和失败
SUCCESS=$(grep -l "\"success\":true" "$TEMP_DIR"/*.txt 2>/dev/null | wc -l | tr -d ' ')
FAIL=$(grep -l "\"success\":false" "$TEMP_DIR"/*.txt 2>/dev/null | wc -l | tr -d ' ')

echo "总请求数: $REQUESTS"
echo "成功: $SUCCESS"
echo "失败: $FAIL"
echo "成功率: $(python3 -c "print($SUCCESS / $REQUESTS * 100)")%"
echo ""

# 计算QPS
DURATION=$(python3 -c "print(($END_TIME - $START_TIME) / 1000)")
QPS=$(python3 -c "print($REQUESTS / $DURATION)")
echo "总耗时: ${DURATION}s"
echo "QPS: $QPS"
echo ""

# 显示最新的竞拍价格
echo "【最终竞拍状态】"
curl -s "$BASE_URL/api/v1/auctions/$AUCTION_ID" | python3 -m json.tool

# 清理临时文件
rm -rf "$TEMP_DIR"

echo ""
echo "====================================="
echo "   测试完成"
echo "====================================="
