# 打包需要的文件，并压缩
#!/bin/bash

# 设置默认参数
ARCH=${1:-"amd64"}

# 切换到项目根目录
cd "$(dirname "$0")"

DIR="output/libreoffice-api-${ARCH}"
# 将docker-compose.yml文件复制到目录里
cp docker-compose.yml ${DIR}/docker-compose.yml
# 将deploy.sh文件复制到目录里
cp deploy.sh ${DIR}/
# 将stop.sh文件复制到目录里
cp stop.sh ${DIR}/

cd output

# 删除原本的压缩文件
rm -f libreoffice-api-${ARCH}.tar.gz

# 压缩文件
# tar -cJvf agentsgo-${ARCH}.tar.xz agentsgo-${ARCH}
tar -czvf libreoffice-api-${ARCH}.tar.gz libreoffice-api-${ARCH}
# 输出压缩文件的大小
du -sh libreoffice-api-${ARCH}.tar.gz