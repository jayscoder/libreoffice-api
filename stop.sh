# 检测docker compose命令版本
if command -v docker-compose &> /dev/null; then
    # 使用旧版docker-compose
    echo "使用 docker-compose 命令..."
    docker-compose -f docker-compose.yml down
else
    # 使用新版docker compose
    echo "使用 docker compose 命令..."
    docker compose -f docker-compose.yml down
fi