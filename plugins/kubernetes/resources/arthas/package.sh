#!/bin/bash

# Arthas 打包脚本
# 用于创建预打包的 Arthas tar 文件

set -e

ARTHAS_VERSION="3.7.2"
OUTPUT_DIR="$(pwd)/plugins/kubernetes/resources/arthas"
TEMP_DIR=$(mktemp -d)

echo "正在下载 Arthas ${ARTHAS_VERSION}..."

# 下载 Arthas
cd "$TEMP_DIR"
curl -sL "https://arthas.aliyun.com/arthas-boot.jar" -o arthas-boot.jar

# 创建目录结构
mkdir -p arthas-bin

# 启动 Arthas 一次以下载必要的 jar 包（后台运行）
java -jar arthas-boot.jar --target-ip 127.0.0.1 &
ARTHAS_PID=$!

# 等待 Arthas 下载依赖
sleep 10

# 停止 Arthas
kill $ARTHAS_PID 2>/dev/null || true
sleep 2

# 将 arthas-boot.jar 复制到输出目录
cp arthas-boot.jar arthas-bin/v


# 打包
tar -czf "$OUTPUT_DIR/arthas-bin.tar.gz" arthas-bin/

# 清理
rm -rf "$TEMP_DIR"

echo "✅ Arthas 打包完成: $OUTPUT_DIR/arthas-bin.tar.gz"
echo "文件大小: $(du -h "$OUTPUT_DIR/arthas-bin.tar.gz" | cut -f1)"
