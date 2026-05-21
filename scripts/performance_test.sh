#!/bin/bash

# 直播竞拍系统性能测试脚本
# 使用 Apache Bench (ab) 进行压力测试

echo "====================================="
echo "   直播竞拍系统性能测试"
echo "====================================="
echo ""

# 检查服务状态
echo "【1】检查服务状态..."
curl -s http://localhost:8081/health > /dev/null && echo "✅ Product Service (8081)" || echo "❌ Product Service (8081)"
curl -s http://localhost:8082/health > /dev/null && echo "✅ Auction Service (8082)" || echo "❌ Auction Service (8082)"
curl -s http://localhost:8083/health > /dev/null && echo "✅ WebSocket Service (8083)" || echo "❌ WebSocket Service (8083)"
echo ""

# 测试参数
AUCTION_ID=3
BASE_URL="http://localhost:8082"
REQUESTS=1000
CONCURRENCY=100

# 测试1: 商品列表API
echo "【2】商品列表API性能测试"
echo "-------------------------------------"
echo "URL: $BASE_URL/api/v1/products"
echo "请求数: $REQUESTS"
echo "并发数: $CONCURRENCY"
ab -n $REQUESTS -c $CONCURRENCY "$BASE_URL/api/v1/products" 2>&1 | grep -E "(Requests per second|Time per request|Transfer rate|Failed requests)"
echo ""

# 测试2: 竞拍详情API
echo "【3】竞拍详情API性能测试"
echo "-------------------------------------"
echo "URL: $BASE_URL/api/v1/auctions/$AUCTION_ID"
ab -n $REQUESTS -c $CONCURRENCY "$BASE_URL/api/v1/auctions/$AUCTION_ID" 2>&1 | grep -E "(Requests per second|Time per request|Transfer rate|Failed requests)"
echo ""

# 测试3: 排名查询API
echo "【4】排名查询API性能测试"
echo "-------------------------------------"
echo "URL: $BASE_URL/api/v1/auctions/$AUCTION_ID/ranking"
ab -n $REQUESTS -c $CONCURRENCY "$BASE_URL/api/v1/auctions/$AUCTION_ID/ranking" 2>&1 | grep -E "(Requests per second|Time per request|Transfer rate|Failed requests)"
echo ""

# 测试4: 并发出价测试（使用自定义脚本）
echo "【5】并发出价测试"
echo "-------------------------------------"
echo "使用Go性能测试脚本进行并发出价测试..."
echo "请运行: go run scripts/performance_test.go"
echo ""

# 测试5: WebSocket连接测试说明
echo "【6】WebSocket连接测试"
echo "-------------------------------------"
echo "WebSocket性能测试需要专门工具，推荐使用："
echo ""
echo "1. wscat (单个连接测试):"
echo "   npm install -g wscat"
echo "   wscat -c 'ws://localhost:8083/ws?auction_id=$AUCTION_ID&user_id=1'"
echo ""
echo "2. Artillery (压力测试):"
echo "   npm install -g artillery"
echo "   artillery run scripts/websocket-load-test.yml"
echo ""
echo "3. k6 (专业性能测试):"
echo "   k6 run scripts/websocket-test.js"
echo ""

echo "====================================="
echo "   性能测试完成"
echo "====================================="
