#!/bin/bash

# 直播竞拍系统 - 快速部署脚本
# 用法: ./deploy.sh [环境] [域名]
# 示例: ./deploy.sh production auction.example.com

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 检查依赖
check_dependencies() {
    print_info "检查系统依赖..."

    local missing_deps=()

    if ! command_exists node; then
        missing_deps+=("node")
    fi

    if ! command_exists npm; then
        missing_deps+=("npm")
    fi

    if ! command_exists git; then
        missing_deps+=("git")
    fi

    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_error "缺少以下依赖: ${missing_deps[*]}"
        print_info "请先安装这些依赖后再运行部署脚本"
        exit 1
    fi

    print_success "所有依赖已安装"
}

# 获取参数
ENVIRONMENT=${1:-production}
DOMAIN=${2:-localhost}

print_info "部署环境: $ENVIRONMENT"
print_info "域名: $DOMAIN"

# 检查依赖
check_dependencies

# 检查Node.js版本
NODE_VERSION=$(node -v | cut -d 'v' -f 2 | cut -d '.' -f 1)
if [ "$NODE_VERSION" -lt 18 ]; then
    print_error "Node.js版本过低，需要18+，当前版本: $(node -v)"
    exit 1
fi

print_success "Node.js版本检查通过: $(node -v)"

# 安装依赖
print_info "安装项目依赖..."
npm install
print_success "依赖安装完成"

# 创建环境变量文件
print_info "配置环境变量..."

if [ ! -f ".env.$ENVIRONMENT" ]; then
    print_warning ".env.$ENVIRONMENT 文件不存在，创建默认配置"

    if [ "$ENVIRONMENT" = "production" ]; then
        cat > .env.production <<EOF
# API配置
VITE_API_BASE_URL=https://api.$DOMAIN
VITE_WS_BASE_URL=wss://ws.$DOMAIN

# 应用配置
VITE_APP_TITLE=直播竞拍系统
NODE_ENV=production
EOF
    elif [ "$ENVIRONMENT" = "staging" ]; then
        cat > .env.staging <<EOF
# API配置
VITE_API_BASE_URL=https://api-staging.$DOMAIN
VITE_WS_BASE_URL=wss://ws-staging.$DOMAIN

# 应用配置
VITE_APP_TITLE=直播竞拍系统 - 测试环境
NODE_ENV=staging
EOF
    else
        cat > .env.development <<EOF
# API配置
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_BASE_URL=ws://localhost:8080

# 应用配置
VITE_APP_TITLE=直播竞拍系统 - 开发环境
NODE_ENV=development
EOF
    fi
fi

print_success "环境变量配置完成"

# 运行测试
print_info "运行测试..."
if npm test; then
    print_success "测试通过"
else
    print_warning "测试失败，是否继续部署？(y/n)"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        print_info "部署已取消"
        exit 1
    fi
fi

# 构建项目
print_info "构建项目..."
if [ "$ENVIRONMENT" = "production" ]; then
    npm run build
else
    npm run build -- --mode $ENVIRONMENT
fi

if [ $? -eq 0 ]; then
    print_success "构建完成"
else
    print_error "构建失败"
    exit 1
fi

# 检查构建产物
if [ ! -d "dist" ]; then
    print_error "构建产物目录不存在"
    exit 1
fi

print_success "构建产物检查通过"

# 显示构建信息
print_info "构建信息:"
echo "  - 文件数量: $(find dist -type f | wc -l)"
echo "  - 总大小: $(du -sh dist | cut -f1)"
echo "  - 主要文件:"
ls -lh dist/assets/*.{js,css} 2>/dev/null | awk '{print "    - " $9 " (" $5 ")"}'

# 询问是否部署到服务器
print_info "是否部署到远程服务器？(y/n)"
read -r deploy_response

if [[ "$deploy_response" =~ ^[Yy]$ ]]; then
    print_info "请输入服务器信息:"
    read -p "服务器地址: " SERVER_HOST
    read -p "用户名: " SERVER_USER
    read -p "部署路径 [默认: /var/www/auction-h5]: " DEPLOY_PATH
    DEPLOY_PATH=${DEPLOY_PATH:-/var/www/auction-h5}

    # 测试SSH连接
    print_info "测试SSH连接..."
    if ssh -o ConnectTimeout=5 "$SERVER_USER@$SERVER_HOST" "echo '连接成功'" 2>/dev/null; then
        print_success "SSH连接正常"
    else
        print_error "SSH连接失败，请检查服务器地址和SSH密钥"
        exit 1
    fi

    # 备份现有版本
    print_info "备份现有版本..."
    ssh "$SERVER_USER@$SERVER_HOST" "if [ -d $DEPLOY_PATH ]; then \
        mkdir -p $DEPLOY_PATH/backups && \
        tar -czf $DEPLOY_PATH/backups/backup-\$(date +%Y%m%d_%H%M%S).tar.gz -C $DEPLOY_PATH dist 2>/dev/null || true; \
    fi"
    print_success "备份完成"

    # 创建部署目录
    print_info "创建部署目录..."
    ssh "$SERVER_USER@$SERVER_HOST" "mkdir -p $DEPLOY_PATH/dist"
    print_success "目录创建完成"

    # 上传文件
    print_info "上传构建产物..."
    rsync -avz --delete dist/ "$SERVER_USER@$SERVER_HOST:$DEPLOY_PATH/dist/"
    print_success "文件上传完成"

    # 设置权限
    print_info "设置文件权限..."
    ssh "$SERVER_USER@$SERVER_HOST" "sudo chown -R www-data:www-data $DEPLOY_PATH && \
        sudo chmod -R 755 $DEPLOY_PATH"
    print_success "权限设置完成"

    # 重载Nginx
    print_info "重载Nginx配置..."
    ssh "$SERVER_USER@$SERVER_HOST" "sudo nginx -t && sudo systemctl reload nginx"
    print_success "Nginx重载完成"

    print_success "🎉 部署成功！"
    print_info "访问地址: http://$DOMAIN"

else
    print_info "本地构建已完成，构建产物位于 dist/ 目录"
    print_info "您可以手动将 dist/ 目录上传到服务器"

    # 本地预览选项
    print_info "是否在本地预览构建结果？(y/n)"
    read -r preview_response

    if [[ "$preview_response" =~ ^[Yy]$ ]]; then
        print_info "启动本地预览服务器..."
        npm run preview
    fi
fi

# 生成部署报告
REPORT_FILE="deployment-report-$(date +%Y%m%d_%H%M%S).txt"
cat > "$REPORT_FILE" <<EOF
直播竞拍系统 - 部署报告
========================

部署时间: $(date)
部署环境: $ENVIRONMENT
域名: $DOMAIN
Node.js版本: $(node -v)
npm版本: $(npm -v)

构建信息:
- 文件数量: $(find dist -type f | wc -l)
- 总大小: $(du -sh dist | cut -f1)

主要文件:
$(ls -lh dist/assets/*.{js,css} 2>/dev/null)

部署状态: 成功 ✅
EOF

print_success "部署报告已生成: $REPORT_FILE"

print_info "下一步建议:"
echo "  1. 检查应用功能是否正常"
echo "  2. 监控应用性能和错误日志"
echo "  3. 配置SSL证书（如果使用HTTP/2）"
echo "  4. 设置备份策略"
echo "  5. 配置监控告警"

print_success "部署流程完成！ 🚀"
