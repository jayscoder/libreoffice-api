# LibreOffice 文档转换 API (Go 版本)

这是一个基于 Gin 和 LibreOffice 的文档转换 API 服务，可以将各种格式的文档转换为其他格式，如 TXT、PDF、DOCX 等。此版本使用 Go 语言重写了原本的 Python 版本。

## 功能特点

- 支持多种文档格式的转换（DOC, DOCX, XLS, XLSX, PPT, PPTX, PDF 等）
- 基于 LibreOffice 的强大转换功能
- 文档转换后提供下载链接
- 支持配置文件保存期限，自动清理过期文件
- RESTful API 设计

## 系统要求

- Go 1.18+ (推荐使用 Go 1.20+)
- LibreOffice 安装并可从命令行访问
- 足够的存储空间用于保存转换后的文档

## 安装与配置

### 安装依赖项

```bash
# 安装 LibreOffice（以 Ubuntu 为例）
sudo apt-get update
sudo apt-get install -y libreoffice

# 克隆项目
git clone https://your-repository/libreoffice-api.git
cd libreoffice-api

# 安装 Go 依赖
go mod tidy
```

### 环境变量配置

创建 `.env` 文件，设置以下环境变量：

```
DEBUG=true                # 是否启用调试模式
MAX_CONTENT_LENGTH=104857600  # 允许上传的最大文件大小（字节），默认 100MB
SOFFICE_PATH=soffice      # LibreOffice 可执行文件路径
FILE_EXPIRY_HOURS=24      # 文件过期时间（小时），设置为 -1 表示永不过期
```

## 构建与运行

### 本地运行

```bash
# 运行开发服务器
go run main.go

# 或者构建后运行
go build -o libreoffice-api
./libreoffice-api
```

服务默认在 `http://localhost:15000` 上运行。

### 使用 Docker 构建与运行

```bash
# 构建 Docker 镜像
docker build -t libreoffice-api:go .

# 运行 Docker 容器
docker run -d -p 15000:15000 -v ./data:/app/data -v ./tmp:/app/tmp --name libreoffice-api libreoffice-api:go
```

## API 接口说明

### 1. 文档转换 API

**接口**：`POST /convert`

**请求参数**：

- `file`：要转换的文件（multipart/form-data）
- `format`：转换目标格式，默认为 `txt`

**支持的格式**：

- 文本格式：`txt`
- PDF 格式：`pdf`
- Word 格式：`docx`
- Excel 格式：`xlsx`
- HTML 格式：`html`
- 更多格式参考 LibreOffice 文档

**响应示例**：

```json
{
  "success": true,
  "filename": "原始文件名.docx",
  "download_url": "http://localhost:15000/download/20231201/文件名_1701410000000.pdf",
  "download_filename": "20231201/文件名_1701410000000.pdf",
  "expiry": "2023-12-02 10:00:00"
}
```

### 2. 文件下载 API

**接口**：`GET /download/<path:filename>`

**响应**：转换后的文件内容

### 3. 健康检查 API

**接口**：`GET /health`

**响应示例**：

```json
{
  "status": "healthy",
  "libreoffice": true,
  "version": "LibreOffice 7.5.3",
  "data_dir": "/app/data",
  "file_expiry_hours": 24
}
```

## 文件存储结构

转换后的文件会保存在以下格式的目录中：

```
DATA_DIR/yyyymmdd/[原始文件名]_[时间戳].[扩展名]
```

例如：

```
data/20231201/example_1701410000000.pdf
```

## 注意事项

1. 文件会在指定的过期时间后自动删除，除非设置为永不过期
2. 服务需要足够的磁盘空间用于存储转换后的文件
3. 较大文件的转换可能需要更多时间
4. 确保 LibreOffice 正确安装并可通过命令行访问

## 与 Python 版本的区别

1. 使用 Go 语言的并发特性，性能更佳
2. 简化了代码结构，移除了 Swagger 文档依赖
3. 使用 Gin 框架提供 HTTP 服务
4. 文件清理和处理机制更加高效

## 贡献

欢迎提交问题报告和改进建议！
