#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║      可观测性平台 - 启动脚本                             ║${NC}"
echo -e "${BLUE}║      (Loki + Prometheus + Grafana)                       ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}错误: Docker 未运行，请先启动 Docker${NC}"
    exit 1
fi

case "$1" in
    start)
        echo -e "${YELLOW}启动服务...${NC}"
        cd "$SCRIPT_DIR"
        docker compose up -d

        echo ""
        echo -e "${GREEN}✓ 服务启动完成!${NC}"
        echo ""
        echo -e "${YELLOW}访问地址:${NC}"
        echo -e "  ├─ Grafana:    http://localhost:3002 (admin/admin)"
        echo -e "  ├─ Prometheus: http://localhost:9090"
        echo -e "  └─ Loki:       http://localhost:3100"
        echo ""
        echo -e "${YELLOW}预置仪表板:${NC}"
        echo -e "  ├─ 业务监控仪表板 (直播进入/成交次数/支付统计)"
        echo -e "  └─ 微服务日志仪表板 (全链路日志查询)"
        echo ""
        echo -e "${YELLOW}常用命令:${NC}"
        echo "  ./start.sh stop      # 停止服务"
        echo "  ./start.sh restart   # 重启服务"
        echo "  ./start.sh logs      # 查看日志"
        echo "  ./start.sh status    # 查看状态"
        ;;

    stop)
        echo -e "${YELLOW}停止服务...${NC}"
        cd "$SCRIPT_DIR"
        docker compose down
        echo -e "${GREEN}✓ 服务已停止${NC}"
        ;;

    restart)
        echo -e "${YELLOW}重启服务...${NC}"
        cd "$SCRIPT_DIR"
        docker compose restart
        echo -e "${GREEN}✓ 服务已重启${NC}"
        ;;

    logs)
        cd "$SCRIPT_DIR"
        if [ -z "$2" ]; then
            docker compose logs -f --tail=100
        else
            docker compose logs -f --tail=100 "$2"
        fi
        ;;

    status)
        echo -e "${YELLOW}服务状态:${NC}"
        cd "$SCRIPT_DIR"
        docker compose ps
        ;;

    clean)
        echo -e "${RED}警告: 这将删除所有数据!${NC}"
        read -p "确定要继续吗? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            cd "$SCRIPT_DIR"
            docker compose down -v
            echo -e "${GREEN}✓ 清理完成${NC}"
        fi
        ;;

    *)
        echo "用法: $0 {start|stop|restart|logs|status|clean}"
        echo ""
        echo -e "${YELLOW}命令说明:${NC}"
        echo "  start   - 启动所有服务 (Loki + Prometheus + Grafana)"
        echo "  stop    - 停止所有服务"
        echo "  restart - 重启所有服务"
        echo "  logs    - 查看服务日志"
        echo "  status  - 查看服务状态"
        echo "  clean   - 清理所有数据(包括存储)"
        echo ""
        echo -e "${YELLOW}组件说明:${NC}"
        echo "  Loki       - 日志聚合存储"
        echo "  Prometheus - 指标收集存储"
        echo "  Grafana    - 统一可视化界面"
        exit 1
        ;;
esac
