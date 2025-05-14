# LibreOffice 文档转换 API (Go 版本)

这是一个基于 Gin 和 LibreOffice 的文档转换 API 服务，可以将各种格式的文档转换为其他格式，如 TXT、PDF、DOCX 等。该项目是原 Python 版本的 Go 重写版，保留了所有核心功能并提供了性能和可靠性方面的改进。

## 功能特点

- **多格式支持**：转换各种文档格式（DOC, DOCX, XLS, XLSX, PPT, PPTX, PDF 等）
- **高效处理**：基于 Go 的并发特性，处理大量转换请求更高效
- **简单集成**：RESTful API 设计，易于与其他系统集成
- **自动清理**：支持配置文件保存期限，自动清理过期文件
- **中间格式转换**：解决了 PDF 到 TXT 等特殊转换路径问题
- **友好界面**：提供直观的 Web 界面进行测试和文档查看

## 系统要求

- Go 1.18+（推荐 Go 1.20+）
- LibreOffice 安装并可从命令行访问
- 足够的存储空间用于保存转换后的文档

## 安装与配置

### 环境变量配置

创建 `.env` 文件，设置以下环境变量：

```
DEBUG=true                 # 是否启用调试模式
MAX_CONTENT_LENGTH=104857600  # 允许上传的最大文件大小（字节），默认 100MB
SOFFICE_PATH=soffice        # LibreOffice 可执行文件路径
FILE_EXPIRY_HOURS=24        # 文件过期时间（小时），设置为 -1 表示永不过期
```

## 构建与运行

### 本地运行

```bash
# 安装依赖
go mod tidy

# 运行开发服务器
go run main.go

# 或者构建后运行
go build -o libreoffice-api-go
./libreoffice-api-go
```

服务默认在 `http://localhost:15000` 上运行。

### 使用 Docker 运行

```bash
# 构建Docker镜像
./build_go.sh amd64  # 构建AMD64架构的镜像
# 或
./build_go.sh arm64  # 构建ARM64架构的镜像

# 运行Docker容器
docker run -d -p 15000:15000 -v ./data:/app/data -v ./tmp:/app/tmp --name libreoffice-api-go libreoffice-api-go:latest
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

**接口**：`GET /download/:filename`

**请求参数**：

- `filename`：文件路径，格式为 `日期/文件名`，例如 `20231201/文件名_1701410000000.pdf`

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

## 主要改进与特点

相比 Python 版本，Go 实现的主要改进包括：

1. **信号处理与优雅退出**：添加了信号处理，确保服务可以优雅关闭，不丢失数据
2. **并发处理**：利用 Go 的 goroutine 处理文件清理等后台任务
3. **中间格式转换**：针对 PDF 到 TXT 等特殊转换路径，添加了中间格式转换步骤
4. **更完善的错误处理**：捕获并处理更多潜在错误，提高服务稳定性
5. **友好的 Web 界面**：提供更完善的首页，包含 API 文档和转换测试表单

## 文件结构

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
5. 某些格式之间的转换可能无法保留全部格式和内容
