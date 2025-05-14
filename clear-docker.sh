# 清理Docker缓存以释放空间
echo "清理Docker系统以释放空间..."
docker system prune -f
docker builder prune -f
docker image prune -f