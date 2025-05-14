# Docker Compose 部署说明

本文档介绍如何使用 Docker Compose 来部署和管理 LibreOffice API 服务。

## 先决条件

- 安装 [Docker](https://docs.docker.com/get-docker/)
- 安装 [Docker Compose](https://docs.docker.com/compose/install/)

## 快速开始

1. 在当前目录下创建 `.env` 文件（可选）：

```
# 调试模式
DEBUG=true

# 最大上传文件大小 (100MB)
MAX_CONTENT_LENGTH=104857600
```

2. 启动服务：

```bash
docker compose up -d
```

服务将在后台启动，并可通过 http://localhost:15000 访问。API 文档可通过 http://localhost:15000/docs/ 访问。

## 配置说明

所有配置都可以通过 `.env` 文件或环境变量来设置：

| 环境变量             | 说明                   | 默认值            |
| -------------------- | ---------------------- | ----------------- |
| `DEBUG`              | 是否启用调试模式       | true              |
| `MAX_CONTENT_LENGTH` | 上传文件最大大小(字节) | 104857600 (100MB) |

## 端口说明

服务暴露以下端口：

| 端口号                  | 说明                   |
| ----------------------- | ---------------------- |
| 15000                   | API 服务端口           |
| 15001 (映射到容器 3000) | LibreOffice UI 接口    |
| 15002 (映射到容器 3001) | LibreOffice 编辑器接口 |

## 常用命令

### 启动服务

```bash
docker compose up -d
```

### 查看日志

```bash
docker compose logs -f
```

### 停止服务

```bash
docker compose down
```

### 重新构建并启动

```bash
docker compose up -d --build
```

## 健康检查

服务包含内置的健康检查，每 30 秒检查一次 `/health` 接口。可以通过以下命令查看服务健康状态：

```bash
docker compose ps
```

## 数据卷

服务定义了一个命名卷 `libreoffice_tmp`，用于存储临时文件。这个卷会被自动创建和管理。
