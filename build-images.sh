#!/bin/bash
# OpsHub 镜像构建脚本
# 使用方法: ./build-images.sh [组织名] [版本号]

set -e

# 配置
SWR_REGION="swr.cn-east-3.myhuaweicloud.com"
SWR_ORG="${1:-opshub}"  # 默认组织名，请修改为你的组织名
VERSION="${2:-${OPSHUB_IMAGE_TAG:-}}"
DOCKER_PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"

if [[ -z "${VERSION}" || "${VERSION}" == "latest" || "${VERSION}" == "dev" ]]; then
    echo "错误: 发布构建必须显式提供不可变版本，例如 ./build-images.sh dyclouds v0.0.10" >&2
    exit 1
fi

if [[ "${ALLOW_DIRTY_BUILD:-0}" != "1" ]]; then
    if ! git diff --quiet || ! git diff --cached --quiet || [[ -n "$(git ls-files --others --exclude-standard | head -n 1)" ]]; then
        echo "错误: 发布构建要求干净工作区，请先提交代码；临时验证可显式设置 ALLOW_DIRTY_BUILD=1" >&2
        exit 1
    fi
fi

SOURCE_REVISION="$(git rev-parse HEAD)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_LABELS=(
    --label "org.opencontainers.image.version=${VERSION}"
    --label "org.opencontainers.image.revision=${SOURCE_REVISION}"
    --label "org.opencontainers.image.created=${BUILD_DATE}"
    --label "org.opencontainers.image.source=https://github.com/ydcloud-dy/opshub"
)

HOST_ARCH="${DOCKER_BUILD_ARCH:-$(docker info --format '{{.Architecture}}' 2>/dev/null || uname -m)}"
case "${HOST_ARCH}" in
    amd64|x86_64) HOST_ARCH="amd64" ;;
    arm64|aarch64) HOST_ARCH="arm64" ;;
esac
DOCKER_BUILD_PLATFORM="${DOCKER_BUILD_PLATFORM:-linux/${HOST_ARCH}}"
TARGET_OS="${DOCKER_PLATFORM%%/*}"
TARGET_ARCH="${DOCKER_PLATFORM#*/}"
TARGET_ARCH="${TARGET_ARCH%%/*}"

BACKEND_IMAGE="${SWR_REGION}/${SWR_ORG}/opshub-backend:${VERSION}"
FRONTEND_IMAGE="${SWR_REGION}/${SWR_ORG}/opshub-frontend:${VERSION}"
LOG_GATEWAY_IMAGE="${SWR_REGION}/${SWR_ORG}/opshub-log-gateway:${VERSION}"
LOG_WRITER_IMAGE="${SWR_REGION}/${SWR_ORG}/opshub-log-writer:${VERSION}"
LOG_AGENT_IMAGE="${SWR_REGION}/${SWR_ORG}/opshub-log-agent:${VERSION}"
LOG_AGENT_VERSION="${LOG_AGENT_VERSION:-0.3.0}"

echo "================================================"
echo "OpsHub 镜像构建"
echo "================================================"
echo "后端镜像: ${BACKEND_IMAGE}"
echo "前端镜像: ${FRONTEND_IMAGE}"
echo "Log Gateway 镜像: ${LOG_GATEWAY_IMAGE}"
echo "Log Writer 镜像: ${LOG_WRITER_IMAGE}"
echo "Log Agent 镜像: ${LOG_AGENT_IMAGE}"
echo "目标平台: ${DOCKER_PLATFORM}"
echo "构建平台: ${DOCKER_BUILD_PLATFORM}"
echo "================================================"

# 构建后端镜像
echo ""
echo "🔨 构建后端镜像..."
docker build --platform "${DOCKER_PLATFORM}" "${BUILD_LABELS[@]}" -t "${BACKEND_IMAGE}" -f Dockerfile .

# 构建前端镜像
echo ""
echo "🔨 构建前端镜像..."
docker build --platform "${DOCKER_PLATFORM}" "${BUILD_LABELS[@]}" -t "${FRONTEND_IMAGE}" -f Dockerfile.frontend .

# 构建日志数据面镜像
echo ""
echo "🔨 构建 Log Gateway 镜像..."
docker build --platform "${DOCKER_PLATFORM}" \
    "${BUILD_LABELS[@]}" \
    --build-arg BUILDPLATFORM="${DOCKER_BUILD_PLATFORM}" \
    --build-arg TARGETPLATFORM="${DOCKER_PLATFORM}" \
    --build-arg TARGETOS="${TARGET_OS}" \
    --build-arg TARGETARCH="${TARGET_ARCH}" \
    -t "${LOG_GATEWAY_IMAGE}" -f Dockerfile.log-gateway .

echo ""
echo "🔨 构建 Log Writer 镜像..."
docker build --platform "${DOCKER_PLATFORM}" \
    "${BUILD_LABELS[@]}" \
    --build-arg BUILDPLATFORM="${DOCKER_BUILD_PLATFORM}" \
    --build-arg TARGETPLATFORM="${DOCKER_PLATFORM}" \
    --build-arg TARGETOS="${TARGET_OS}" \
    --build-arg TARGETARCH="${TARGET_ARCH}" \
    -t "${LOG_WRITER_IMAGE}" -f Dockerfile.log-writer .

echo ""
echo "构建 Log Agent 镜像..."
docker build --platform "${DOCKER_PLATFORM}" \
    "${BUILD_LABELS[@]}" \
    --build-arg BUILDPLATFORM="${DOCKER_BUILD_PLATFORM}" \
    --build-arg TARGETPLATFORM="${DOCKER_PLATFORM}" \
    --build-arg TARGETOS="${TARGET_OS}" \
    --build-arg TARGETARCH="${TARGET_ARCH}" \
    --build-arg AGENT_VERSION="${LOG_AGENT_VERSION}" \
    -t "${LOG_AGENT_IMAGE}" -f Dockerfile.log-agent .

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
echo "docker push ${LOG_GATEWAY_IMAGE}"
echo "docker push ${LOG_WRITER_IMAGE}"
echo "docker push ${LOG_AGENT_IMAGE}"
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
echo "logCenter:"
echo "  ingest:"
echo "    gateway:"
echo "      image:"
echo "        repository: ${SWR_REGION}/${SWR_ORG}/opshub-log-gateway"
echo "        tag: ${VERSION}"
echo "    writer:"
echo "      image:"
echo "        repository: ${SWR_REGION}/${SWR_ORG}/opshub-log-writer"
echo "        tag: ${VERSION}"
echo "  kubernetesCollector:"
echo "    image: ${LOG_AGENT_IMAGE}"
echo ""
