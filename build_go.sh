#!/bin/bash

# 设置默认参数
ARCH=${1:-"amd64"}
IMAGE_NAME="libreoffice-api-go:latest"
PLATFORM="linux/${ARCH}"

# 获取脚本所在目录
SCRIPT_DIR=$(dirname "$0")
cd "${SCRIPT_DIR}"
LIBREOFFICE_DIR=$(pwd)

echo "===== 开始构建 LibreOffice API (Go版本) 镜像 (${ARCH}) ====="

# 确保输出目录存在
OUTPUT_DIR=${LIBREOFFICE_DIR}/output/
mkdir -p ${OUTPUT_DIR}/

# 确保tmp目录存在
mkdir -p ${LIBREOFFICE_DIR}/tmp/

# 构建镜像
echo "构建 LibreOffice API (Go版本) 镜像 (${ARCH})..."
for attempt in {1..3}; do
    echo "尝试构建 LibreOffice API (Go版本) 镜像 (第${attempt}次)..."
    docker buildx build \
        --platform ${PLATFORM} \
        --build-arg TARGETPLATFORM=${PLATFORM} \
        --build-arg TARGETARCH=${ARCH} \
        -t ${IMAGE_NAME} \
        -f ${LIBREOFFICE_DIR}/Dockerfile.golang \
        --load ${LIBREOFFICE_DIR} && break
        
    if [ $attempt -lt 3 ]; then
        echo "构建失败，2秒后重试..."
        sleep 2
    fi
done

# 检查构建结果
if [ $? -ne 0 ]; then
    echo "LibreOffice API (Go版本) 镜像构建失败!"
    exit 1
fi

# 保存镜像
echo "保存 LibreOffice API (Go版本) ${ARCH} 版本镜像..."
if ! docker save ${IMAGE_NAME} > ${OUTPUT_DIR}/libreoffice-api-go-linux_${ARCH}.tar.gz; then
    echo "错误: 保存镜像失败，可能是磁盘空间不足"
    exit 1
fi

echo "===== LibreOffice API (Go版本) 镜像构建完成 ====="
echo "镜像已保存至: ${OUTPUT_DIR}/libreoffice-api-go-linux_${ARCH}.tar.gz"
echo ""
echo "使用以下命令来加载和运行镜像:"
echo "docker load < ${OUTPUT_DIR}/libreoffice-api-go-linux_${ARCH}.tar.gz"
echo "docker run -d -p 15000:15000 -v \$(pwd)/data:/app/data -v \$(pwd)/tmp:/app/tmp --name libreoffice-api-go ${IMAGE_NAME}" 