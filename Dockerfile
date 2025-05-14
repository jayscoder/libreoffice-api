FROM docker-0.unsee.tech/linuxserver/libreoffice
# 设置 Alpine Linux 使用国内镜像源
RUN echo "http://mirrors.aliyun.com/alpine/v$(cut -d'.' -f1,2 /etc/alpine-release)/main/" > /etc/apk/repositories && \
    echo "http://mirrors.aliyun.com/alpine/v$(cut -d'.' -f1,2 /etc/alpine-release)/community/" >> /etc/apk/repositories
    
# 安装 Python 和依赖
RUN apk update && apk add --no-cache python3 py3-pip python3-dev

WORKDIR /app

# 创建 API 脚本
COPY ./api.py /app/
COPY ./requirements.txt /app/

# 创建数据和临时目录
RUN mkdir -p /app/data
RUN mkdir -p /app/tmp

# 打印python 的版本
RUN python3 --version
RUN pip3 --version
RUN python3 -m venv /app/venv
RUN source /app/venv/bin/activate
# 使用国内镜像源配置pip
RUN pip3 config set global.index-url https://mirrors.aliyun.com/pypi/simple/

# 安装依赖，增加重试机制
RUN pip3 install --no-cache-dir -r /app/requirements.txt

# 创建临时目录
VOLUME /app/data
VOLUME /app/tmp

# 设置环境变量
ENV PYTHONUNBUFFERED=1
ENV SOFFICE_PATH=/usr/bin/soffice

# 暴露端口
EXPOSE 15000 3000 3001

# 启动服务
CMD ["python3", "/app/api.py"]