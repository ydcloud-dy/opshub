#!/bin/bash
# 下载 GeoLite2-City.mmdb IP地理位置数据库
# 用于 Nginx 插件的 IP 地理位置解析功能

set -e

GEOIP_DIR="${1:-./data}"
GEOIP_FILE="$GEOIP_DIR/GeoLite2-City.mmdb"

# 下载源（按优先级排列）
DOWNLOAD_URLS=(
    "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"
    "https://git.io/GeoLite2-City.mmdb"
    "https://cdn.jsdelivr.net/gh/P3TERX/GeoLite.mmdb@download/GeoLite2-City.mmdb"
)

echo "==> 检查 GeoLite2-City.mmdb..."

# 如果文件已存在且大于 10MB，跳过下载
if [ -f "$GEOIP_FILE" ]; then
    FILE_SIZE=$(stat -f%z "$GEOIP_FILE" 2>/dev/null || stat -c%s "$GEOIP_FILE" 2>/dev/null || echo "0")
    if [ "$FILE_SIZE" -gt 10000000 ]; then
        echo "==> GeoLite2-City.mmdb 已存在 ($(($FILE_SIZE / 1024 / 1024))MB)，跳过下载"
        exit 0
    fi
fi

# 创建目录
mkdir -p "$GEOIP_DIR"

echo "==> 开始下载 GeoLite2-City.mmdb..."

# 尝试从多个源下载
for URL in "${DOWNLOAD_URLS[@]}"; do
    echo "==> 尝试从 $URL 下载..."
    if curl -fSL --connect-timeout 30 --max-time 300 -o "$GEOIP_FILE" "$URL"; then
        # 验证文件大小
        FILE_SIZE=$(stat -f%z "$GEOIP_FILE" 2>/dev/null || stat -c%s "$GEOIP_FILE" 2>/dev/null || echo "0")
        if [ "$FILE_SIZE" -gt 10000000 ]; then
            echo "==> 下载成功！文件大小: $(($FILE_SIZE / 1024 / 1024))MB"
            exit 0
        else
            echo "==> 下载的文件太小，可能不完整，尝试其他源..."
            rm -f "$GEOIP_FILE"
        fi
    else
        echo "==> 从该源下载失败，尝试其他源..."
    fi
done

echo "==> 错误: 无法下载 GeoLite2-City.mmdb"
echo "==> 请手动下载并放置到 $GEOIP_FILE"
exit 1
