#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║      直播竞拍系统 - 统一启动脚本                         ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# 检查 Docker 是否运行
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        echo -e "${RED}错误: Docker 未运行，请先启动 Docker${NC}"
        exit 1
    fi
}

# 启动所有服务
start_all() {
    echo -e "${YELLOW}启动所有服务...${NC}"
    check_docker

    cd "$PROJECT_ROOT"
    docker compose up -d

    echo ""
    echo -e "${GREEN}✓ 服务启动完成!${NC}"
    show_urls
}

# 停止所有服务
stop_all() {
    echo -e "${YELLOW}停止所有服务...${NC}"
    cd "$PROJECT_ROOT"
    docker compose down
    echo -e "${GREEN}✓ 服务已停止${NC}"
}

# 重启所有服务
restart_all() {
    echo -e "${YELLOW}重启所有服务...${NC}"
    cd "$PROJECT_ROOT"
    docker compose restart
    echo -e "${GREEN}✓ 服务已重启${NC}"
}

# 查看服务状态
show_status() {
    echo -e "${YELLOW}服务状态:${NC}"
    cd "$PROJECT_ROOT"
    docker compose ps
}

# 查看服务日志
show_logs() {
    cd "$PROJECT_ROOT"
    if [ -z "$2" ]; then
        docker compose logs -f --tail=100
    else
        docker compose logs -f --tail=100 "$2"
    fi
}

# 显示访问地址
show_urls() {
    echo ""
    echo -e "${BLUE}══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  访问地址${NC}"
    echo -e "${BLUE}══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  ${YELLOW}前端服务${NC}"
    echo -e "  ├─ H5 用户端:    http://localhost:3000"
    echo -e "  └─ Admin 后台:   http://localhost:3001"
    echo ""
    echo -e "  ${YELLOW}后端服务${NC}"
    echo -e "  ├─ Gateway:      http://localhost:8080"
    echo -e "  ├─ Product:      http://localhost:8081"
    echo -e "  ├─ Auction HTTP: http://localhost:8082"
    echo -e "  └─ Auction WS:   ws://localhost:8083"
    echo ""
    echo -e "  ${YELLOW}基础设施${NC}"
    echo -e "  ├─ MySQL:        localhost:3306"
    echo -e "  └─ Redis:        localhost:6379"
    echo ""
    echo -e "  ${YELLOW}日志/监控平台${NC}"
    echo -e "  ├─ Grafana:      http://localhost:3002 (admin/admin)"
    echo -e "  ├─ Prometheus:   http://localhost:9090"
    echo -e "  └─ Loki:         http://localhost:3100"
    echo ""
    echo -e "${BLUE}══════════════════════════════════════════════════════════${NC}"
}

# 仅启动监控平台
start_observability() {
    echo -e "${YELLOW}启动监控平台（日志+指标）...${NC}"
    check_docker

    cd "$PROJECT_ROOT/observability"
    docker compose up -d

    echo ""
    echo -e "${GREEN}✓ 监控平台启动完成!${NC}"
    echo ""
    echo -e "  ${YELLOW}访问地址${NC}"
    echo -e "  ├─ Grafana:    http://localhost:3002 (admin/admin)"
    echo -e "  ├─ Prometheus: http://localhost:9090"
    echo -e "  └─ Loki:       http://localhost:3100"
}

# 仅停止监控平台
stop_observability() {
    echo -e "${YELLOW}停止监控平台...${NC}"
    cd "$PROJECT_ROOT/observability"
    docker compose down
    echo -e "${GREEN}✓ 监控平台已停止${NC}"
}

# 清理所有数据
clean_all() {
    echo -e "${RED}警告: 这将删除所有数据（包括数据库和日志）!${NC}"
    read -p "确定要继续吗? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd "$PROJECT_ROOT"
        docker compose down -v
        echo -e "${GREEN}✓ 清理完成${NC}"
    fi
}

# 帮助信息
show_help() {
    echo "用法: $0 <命令> [参数]"
    echo ""
    echo -e "${YELLOW}命令列表:${NC}"
    echo ""
    echo "  ${GREEN}start${NC}          启动所有服务（应用 + 监控平台）"
    echo "  ${GREEN}stop${NC}           停止所有服务"
    echo "  ${GREEN}restart${NC}        重启所有服务"
    echo "  ${GREEN}status${NC}         查看服务状态"
    echo "  ${GREEN}logs${NC} [服务]    查看服务日志（可指定服务名）"
    echo "  ${GREEN}urls${NC}           显示访问地址"
    echo ""
    echo "  ${BLUE}obs-start${NC}       仅启动监控平台（日志+指标）"
    echo "  ${BLUE}obs-stop${NC}        仅停止监控平台"
    echo ""
    echo "  ${RED}clean${NC}            清理所有数据（危险操作）"
    echo ""
    echo -e "${YELLOW}示例:${NC}"
    echo "  $0 start              # 启动所有服务"
    echo "  $0 logs gateway       # 查看 Gateway 服务日志"
    echo "  $0 obs-start          # 仅启动监控平台"
}

# 主逻辑
case "$1" in
    start)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        restart_all
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs "$@"
        ;;
    urls)
        show_urls
        ;;
    obs-start)
        start_observability
        ;;
    obs-stop)
        stop_observability
        ;;
    clean)
        clean_all
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        show_help
        exit 1
        ;;
esac
