#!/bin/bash
# OpsHub 镜像构建脚本
# 使用方法: ./build-images.sh [组织名] [版本号]

set -e

# 配置
SWR_REGION="swr.cn-east-3.myhuaweicloud.com"
SWR_ORG="${1:-opshub}"  # 默认组织名，请修改为你的组织名
VERSION="${2:-latest}"

BACKEND_IMAGE="${SWR_REGION}/${SWR_ORG}/opshub-backend:${VERSION}"
FRONTEND_IMAGE="${SWR_REGION}/${SWR_ORG}/opshub-frontend:${VERSION}"

echo "================================================"
echo "OpsHub 镜像构建"
echo "================================================"
echo "后端镜像: ${BACKEND_IMAGE}"
echo "前端镜像: ${FRONTEND_IMAGE}"
echo "================================================"

# 构建后端镜像
echo ""
echo "🔨 构建后端镜像..."
docker build -t ${BACKEND_IMAGE} -f Dockerfile .

# 构建前端镜像
echo ""
echo "🔨 构建前端镜像..."
docker build -t ${FRONTEND_IMAGE} -f Dockerfile.frontend .

echo ""
echo "✅ 镜像构建完成！"
echo ""
echo "================================================"
echo "推送镜像到 SWR:"
echo "================================================"
echo ""
echo "# 1. 登录 SWR（首次需要）"
echo "docker login ${SWR_REGION} -u [区域项目名称]@[AK] -p [登录密钥]"
echo ""
echo "# 2. 推送镜像"
echo "docker push ${BACKEND_IMAGE}"
echo "docker push ${FRONTEND_IMAGE}"
echo ""
echo "================================================"
echo "更新 Helm values.yaml:"
echo "================================================"
echo ""
echo "backend:"
echo "  image:"
echo "    repository: ${SWR_REGION}/${SWR_ORG}/opshub-backend"
echo "    tag: ${VERSION}"
echo ""
echo "frontend:"
echo "  image:"
echo "    repository: ${SWR_REGION}/${SWR_ORG}/opshub-frontend"
echo "    tag: ${VERSION}"
echo ""
