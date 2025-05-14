#!/bin/bash

# 测试脚本 - 将data目录中的文件转换为不同格式并保存到output目录

# 设置变量
API_URL="http://localhost:15000/convert"
DATA_DIR="test-data"
OUTPUT_DIR="test-output"

# 创建输出目录
mkdir -p $OUTPUT_DIR

# 可用的转换格式
FORMATS=(
  "txt"
  "pdf"
  "docx"
  "xlsx"
  "html"
)

# 检查API是否在运行
echo "正在检查API服务..."
if ! curl -s "http://localhost:15000/health" > /dev/null; then
  echo "错误：API服务未运行，请先启动API服务"
  exit 1
fi
echo "API服务正常运行"

# 检查数据目录是否存在
if [ ! -d "$DATA_DIR" ]; then
  echo "错误：数据目录 $DATA_DIR 不存在"
  exit 1
fi

# 如果数据目录为空，则提示用户
if [ -z "$(ls -A $DATA_DIR 2>/dev/null)" ]; then
  echo "警告：数据目录 $DATA_DIR 为空，请先添加测试文件"
  exit 1
fi

# 遍历数据目录中的所有文件
echo "开始转换文件..."
echo "----------------------------------------------"

file_count=0
success_count=0

for file in $DATA_DIR/*; do
  # 跳过目录
  if [ -d "$file" ]; then
    continue
  fi
  
  filename=$(basename "$file")
  file_count=$((file_count + 1))
  
  echo "处理文件: $filename"
  
  # 对每个文件尝试所有格式的转换
  for format in "${FORMATS[@]}"; do
    format_name="${format%%:*}"  # 提取格式名称（冒号前的部分）
    
    echo "  转换为 $format_name 格式..."
    
    # 使用curl发送请求
    response=$(curl -s -X POST \
      -F "file=@$file" \
      -F "format=$format" \
      "$API_URL")
    
    # 检查是否成功
    if echo "$response" | grep -q "success\":true"; then
      # 从响应中提取下载链接
      download_url=$(echo "$response" | grep -o '"download_url":"[^"]*"' | sed 's/"download_url":"//;s/"$//')
      
      if [ -n "$download_url" ]; then
        # 创建格式目录
        mkdir -p "$OUTPUT_DIR/$format_name"
        
        # 构建输出文件名
        output_file="$OUTPUT_DIR/$format_name/${filename%.*}_converted.$format_name"
        
        # 下载文件
        echo "  下载转换后的文件到 $output_file"
        curl -s "$download_url" -o "$output_file"
        
        if [ $? -eq 0 ]; then
          echo "  ✓ 转换成功：$output_file"
          success_count=$((success_count + 1))
        else
          echo "  ✗ 下载失败"
        fi
      else
        echo "  ✗ 转换失败：未获取到下载链接"
      fi
    else
      echo "  ✗ 转换失败：$(echo "$response" | grep -o '"error":"[^"]*"' | sed 's/"error":"//;s/"$//')"
    fi
    
    echo ""
  done
  
  echo "----------------------------------------------"
done

# 输出统计信息
echo "转换完成!"
echo "处理文件总数: $file_count"
echo "成功转换数: $success_count"
echo "转换后的文件已保存到 $OUTPUT_DIR 目录"
