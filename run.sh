#!/bin/bash
cd "$(dirname "$0")"

ARCH=${1:-"amd64"}

# 检测 docker compose 命令格式
if docker compose version &>/dev/null; then
    DOCKER_COMPOSE="docker compose"
elif docker-compose --version &>/dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    echo "错误: 未找到 docker compose 或 docker-compose 命令"
    exit 1
fi

# 卸载原来的镜像
$DOCKER_COMPOSE down
docker rm -f libreoffice-api:latest

# 加载镜像
docker load < output/libreoffice-api-linux_${ARCH}.tar.gz

# 运行容器
$DOCKER_COMPOSE up -d

echo "LibreOffice API 服务已启动，访问 http://localhost:15000"