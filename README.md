# LibreOffice 文档转换 API

这是一个基于 Go 和 LibreOffice 的文档转换 API 服务，可以将各种格式的文档转换为其他格式，如 TXT、PDF、DOCX 等。

## 功能特点

- 支持多种文档格式的转换（DOC, DOCX, WPS, TXT, HTML, XML, PDF 等）
- 基于 LibreOffice 的强大转换功能
- 文档转换后提供下载链接
- 支持配置文件保存期限，自动清理过期文件

## 环境变量配置

服务支持通过环境变量进行配置，支持的环境变量包括：

| 环境变量           | 说明                                | 默认值            |
| ------------------ | ----------------------------------- | ----------------- |
| DEBUG              | 是否开启调试模式                    | true              |
| SOFFICE_PATH       | LibreOffice 安装路径                | soffice           |
| MAX_CONTENT_LENGTH | 最大上传文件大小(字节)              | 104857600 (100MB) |
| FILE_EXPIRY_HOURS  | 文件过期时间(小时)，-1 表示永不过期 | 24                |
| PORT               | 服务端口                            | 15000             |

可以通过以下方式配置环境变量：

1. 在项目根目录创建`.env`文件（推荐，项目提供了`env.sample`作为模板）
2. 直接在命令行设置环境变量，例如：`PORT=8080 DEBUG=false ./libreoffice-api`
3. 在 Docker Compose 配置文件中设置

## 构建和运行

### Linux 版本构建 (Docker)

```bash
# 构建 amd64 版本
./build.sh amd64

# 构建 arm64 版本
./build.sh arm64
```

### Linux 版本构建 (不使用 Docker)

```bash
# 使用新脚本构建 Linux 版本
./build_go_linux.sh amd64  # 构建 amd64 版本
./build_go_linux.sh arm64  # 构建 arm64 版本

# 或使用旧脚本
./build_go.sh amd64  # 构建 amd64 版本
./build_go.sh arm64  # 构建 arm64 版本
```

### MacOS 版本构建

```bash
# 使用新脚本构建 MacOS 版本
./build_go_mac.sh arm64  # 适用于 M1/M2 芯片的 Mac
./build_go_mac.sh amd64  # 适用于 Intel 芯片的 Mac
```

### Windows 版本构建

```bash
# 在 Linux/MacOS 上交叉编译 Windows 版本
./build_go_windows.sh amd64  # 构建 Windows x64 版本
./build_go_windows.sh 386    # 构建 Windows x86 版本
```

### 运行

```bash
# 首先复制环境变量模板
cp env.sample .env
# 根据需要修改.env文件

# Linux 版本运行
./output/libreoffice-api-linux-amd64  # 或 libreoffice-api-linux-arm64

# MacOS 版本运行
./output/libreoffice-api-macos-arm64  # M1/M2 Mac
./output/libreoffice-api-macos-amd64  # Intel Mac

# Windows 版本运行
output\libreoffice-api-windows-amd64.exe  # Windows x64
output\libreoffice-api-windows-386.exe    # Windows x86
```

## 注意事项

1. 确保系统已安装 LibreOffice，否则转换功能将无法使用
2. MacOS 上运行可能需要安装 LibreOffice 并正确设置 SOFFICE_PATH 环境变量
3. 在不同操作系统之间构建的二进制文件不能互相运行（例如，Linux 版本不能在 MacOS 上运行，反之亦然）
4. Windows 版本需要在 Windows 环境中运行，并确保 LibreOffice 已安装并添加到系统路径中
