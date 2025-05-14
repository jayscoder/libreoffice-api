# LibreOffice API 使用指南

本文档提供了 LibreOffice 文档转换 API 的常用命令和使用方法参考。

## 快速开始

### 使用 Python 版本

```bash
# 安装依赖
pip install -r requirements.txt

# 启动服务
python api.py
```

### 使用 Go 版本

```bash
# 安装依赖
go mod tidy

# 运行服务
go run main.go

# 或构建后运行
go build -o libreoffice-api-go
./libreoffice-api-go
```

## Docker 部署

### Python 版本

```bash
# 构建镜像
docker build -t libreoffice-api:python .

# 运行容器
docker run -d -p 15000:15000 -v ./data:/app/data --name libreoffice-api-python libreoffice-api:python
```

### Go 版本

```bash
# 构建AMD64架构镜像
./build_go.sh amd64

# 构建ARM64架构镜像
./build_go.sh arm64

# 运行容器
docker run -d -p 15000:15000 -v ./data:/app/data -v ./tmp:/app/tmp --name libreoffice-api-go libreoffice-api-go:latest
```

## API 使用示例

### 文档转换

**使用 curl**

```bash
# 将文档转换为TXT格式
curl -X POST -F "file=@/path/to/document.docx" -F "format=txt" http://localhost:15000/convert

# 将文档转换为PDF格式
curl -X POST -F "file=@/path/to/document.docx" -F "format=pdf" http://localhost:15000/convert
```

**使用 Python**

```python
import requests

url = "http://localhost:15000/convert"
file_path = "/path/to/document.docx"
format = "pdf"  # 目标格式

with open(file_path, "rb") as f:
    files = {"file": f}
    data = {"format": format}
    response = requests.post(url, files=files, data=data)

if response.status_code == 200:
    result = response.json()
    print(f"转换成功，下载链接：{result['download_url']}")
    if "text" in result:
        print(f"文本内容预览：{result['text'][:100]}...")
else:
    print(f"转换失败：{response.json()['error']}")
```

**使用 JavaScript**

```javascript
// 使用fetch API
async function convertDocument(file, format) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("format", format);

  try {
    const response = await fetch("http://localhost:15000/convert", {
      method: "POST",
      body: formData,
    });

    const result = await response.json();

    if (result.success) {
      console.log("转换成功，下载链接：", result.download_url);
      return result;
    } else {
      console.error("转换失败：", result.error);
      throw new Error(result.error);
    }
  } catch (error) {
    console.error("请求出错：", error);
    throw error;
  }
}
```

### 文件下载

```bash
# 下载转换后的文件
curl -O http://localhost:15000/download/20231201/document_1701410000000.pdf
```

### 健康检查

```bash
# 检查服务健康状态
curl http://localhost:15000/health
```

## 支持的格式

LibreOffice 支持多种格式转换，以下是常用格式：

| 源格式         | 目标格式 | 说明                                 |
| -------------- | -------- | ------------------------------------ |
| DOCX, DOC, ODT | TXT      | 提取文本内容                         |
| DOCX, DOC, ODT | PDF      | 转换为 PDF 文档                      |
| XLSX, XLS, ODS | CSV      | 转换为 CSV 表格                      |
| PPTX, PPT, ODP | PDF      | 演示文稿转 PDF                       |
| PDF            | TXT      | 提取 PDF 文本                        |
| PDF            | DOCX     | PDF 转 Word (注意：复杂格式可能丢失) |
| HTML           | PDF      | 网页转 PDF                           |
| RTF            | DOCX     | 富文本转 Word                        |

## 故障排除

1. **文件无法转换**

   - 检查 LibreOffice 是否正确安装：`libreoffice --version`
   - 确认文件格式受支持
   - 查看日志了解详细错误

2. **服务启动问题**

   - Python 版：检查依赖是否完整安装
   - Go 版：确保正确编译

3. **Docker 容器问题**

   - 确保挂载了正确的数据卷
   - 检查容器日志：`docker logs libreoffice-api-go`

4. **文件过大**
   - 调整 MAX_CONTENT_LENGTH 环境变量
