#!/bin/bash

# E2E测试运行脚本

set -e

echo "========================================="
echo "开始运行E2E测试"
echo "========================================="

# 检查Node.js是否安装
if ! command -v node &> /dev/null; then
    echo "错误: 未安装Node.js"
    exit 1
fi

# 检查依赖是否安装
if [ ! -d "node_modules" ]; then
    echo "安装依赖..."
    npm install
fi

# 检查Playwright浏览器是否安装
if ! npx playwright --version &> /dev/null; then
    echo "安装Playwright浏览器..."
    npx playwright install
fi

# 运行测试
echo "运行E2E测试..."
npm run test:e2e

# 生成报告
echo "生成测试报告..."
npm run test:e2e:report

echo "========================================="
echo "测试完成!"
echo "========================================="
