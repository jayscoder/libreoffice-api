#!/bin/bash

# 获取脚本所在目录
SCRIPT_DIR=$(dirname "$0")
cd "${SCRIPT_DIR}"

# 提示用户输入DockerHub用户名和版本
echo "===== 发布 LibreOffice API Docker 镜像到 DockerHub ====="
read -p "DockerHub 用户名 (默认: tongmu): " DOCKER_USERNAME
DOCKER_USERNAME=${DOCKER_USERNAME:-tongmu}

read -p "版本号 (默认: v1.0.0): " VERSION
VERSION=${VERSION:-v1.0.0}

echo "===== 配置信息 ====="
echo "DockerHub 用户名: ${DOCKER_USERNAME}"
echo "版本号: ${VERSION}"
echo "镜像名称: ${DOCKER_USERNAME}/libreoffice-api"
echo ""

# 询问用户是否确认
read -p "确认配置信息? (y/n): " CONFIRM
if [[ "${CONFIRM}" != "y" && "${CONFIRM}" != "Y" ]]; then
    echo "已取消操作"
    exit 1
fi

# 登录DockerHub
echo "===== 登录 DockerHub ====="
docker login -u ${DOCKER_USERNAME}
if [ $? -ne 0 ]; then
    echo "登录失败，退出操作"
    exit 1
fi

# 检查本地镜像
echo "===== 检查本地镜像 ====="
LATEST_IMAGE=$(docker images libreoffice-api:latest -q)
if [ -z "${LATEST_IMAGE}" ]; then
    echo "本地找不到 libreoffice-api:latest 镜像，请先构建镜像"
    exit 1
fi

# 打标签
echo "===== 为镜像打标签 ====="
docker tag libreoffice-api:latest ${DOCKER_USERNAME}/libreoffice-api:latest
docker tag libreoffice-api:latest ${DOCKER_USERNAME}/libreoffice-api:${VERSION}

# 对于amd64架构，添加额外标签
ARCH=$(uname -m)
if [ "${ARCH}" == "x86_64" ]; then
    docker tag libreoffice-api:latest ${DOCKER_USERNAME}/libreoffice-api:${VERSION}-amd64
    echo "添加AMD64架构标签: ${DOCKER_USERNAME}/libreoffice-api:${VERSION}-amd64"
elif [ "${ARCH}" == "arm64" ]; then
    docker tag libreoffice-api:latest ${DOCKER_USERNAME}/libreoffice-api:${VERSION}-arm64
    echo "添加ARM64架构标签: ${DOCKER_USERNAME}/libreoffice-api:${VERSION}-arm64"
fi

# 推送镜像
echo "===== 推送镜像到 DockerHub ====="
docker push ${DOCKER_USERNAME}/libreoffice-api:latest
docker push ${DOCKER_USERNAME}/libreoffice-api:${VERSION}

# 推送架构特定标签
if [ "${ARCH}" == "x86_64" ]; then
    docker push ${DOCKER_USERNAME}/libreoffice-api:${VERSION}-amd64
elif [ "${ARCH}" == "arm64" ]; then
    docker push ${DOCKER_USERNAME}/libreoffice-api:${VERSION}-arm64
fi

echo "===== 发布完成 ====="
echo "镜像已成功发布到 DockerHub"
echo "镜像地址: ${DOCKER_USERNAME}/libreoffice-api"
echo "标签: latest, ${VERSION}"
if [ "${ARCH}" == "x86_64" ]; then
    echo "       ${VERSION}-amd64"
elif [ "${ARCH}" == "arm64" ]; then
    echo "       ${VERSION}-arm64"
fi
echo ""

# 更新docker-compose.yml文件
echo "===== 更新 docker-compose.yml 文件 ====="
sed -i.bak "s|image:.*libreoffice-api.*|image: ${DOCKER_USERNAME}/libreoffice-api:latest|" docker-compose.yml
rm -f docker-compose.yml.bak
echo "docker-compose.yml 已更新，使用 ${DOCKER_USERNAME}/libreoffice-api:latest 镜像"
echo ""

echo "您可以使用以下命令拉取和运行镜像:"
echo "  docker pull ${DOCKER_USERNAME}/libreoffice-api:latest"
echo "  docker-compose up -d" 