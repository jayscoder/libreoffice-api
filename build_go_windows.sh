#!/bin/bash

# 获取脚本所在目录
SCRIPT_DIR=$(dirname "$0")
cd "${SCRIPT_DIR}"
ARCH=${1:-"amd64"}

# 确保输出目录存在
mkdir -p output/bin
OUTPUT_DIR=$(pwd)/output/bin

echo "===== 开始构建 LibreOffice API (Windows ${ARCH}版本) ====="

# 编译Windows版本
CGO_ENABLED=0 GOOS=windows GOARCH=${ARCH} go build -ldflags="-s -w" -o ${OUTPUT_DIR}/libreoffice-api-windows-${ARCH}.exe .

if [ $? -ne 0 ]; then
    echo "编译失败!"
    exit 1
fi

echo "===== LibreOffice API (Windows ${ARCH}版本) 构建完成 ====="
echo "可执行文件已保存至: ${OUTPUT_DIR}/libreoffice-api-windows-${ARCH}.exe"
echo ""
echo "使用以下命令运行 (在Windows系统中):"
echo "${OUTPUT_DIR}/libreoffice-api-windows-${ARCH}.exe" 