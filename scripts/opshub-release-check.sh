#!/usr/bin/env bash

set -Eeuo pipefail

script_directory=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_directory=$(cd -- "${script_directory}/.." && pwd)
chart_directory="${repository_directory}/charts/opshub"
values_file=""
release_name="opshub"
namespace="opshub"
check_compose=false
check_helm=false
strict_mode=false
temporary_files=()

cleanup() {
  for temporary_file in "${temporary_files[@]:-}"; do
    rm -f "${temporary_file}"
  done
}
trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage: scripts/opshub-release-check.sh [options]

Options:
  --compose              Validate docker compose configuration
  --helm                 Validate Helm lint and rendered Kubernetes YAML
  --values FILE          Helm values file used for validation
  --release NAME         Helm release name (default: opshub)
  --namespace NAME       Kubernetes namespace (default: opshub)
  --strict               Fail on placeholder secrets or latest image tags
  -h, --help             Show this help

When no deployment type is selected, both Compose and Helm are checked.
The command only renders and validates configuration; it does not deploy,
restart, scale, or delete any resource.
EOF
}

fail_check() {
  printf 'FAIL: %s\n' "$1" >&2
  return 1
}

warn_check() {
  printf 'WARN: %s\n' "$1" >&2
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail_check "缺少命令: $1"
}

check_compose_config() {
  require_command docker
  local rendered_file
  rendered_file=$(mktemp)
  temporary_files+=("${rendered_file}")
  local compose_arguments=(-f "${repository_directory}/docker-compose.yml")
  if [[ "${strict_mode}" == true ]]; then
    compose_arguments+=(--profile standard)
  fi
  docker compose "${compose_arguments[@]}" config >"${rendered_file}"

  local required_service
  for required_service in mysql redis clickhouse backend frontend log-gateway log-writer; do
    if ! grep -qE "^  ${required_service}:$" "${rendered_file}"; then
      fail_check "Compose 缺少服务: ${required_service}"
    fi
  done
  if ! grep -q 'MYSQL_ROOT_PASSWORD' "${rendered_file}" || ! grep -q 'MYSQL_DATABASE' "${rendered_file}"; then
    fail_check "Compose db-init 未使用运行时数据库配置"
  fi
  if grep -q 'opshub-log-gateway:v0.0.8' "${rendered_file}" && grep -q 'opshub-log-writer:v0.0.8' "${rendered_file}"; then
    warn_check "Compose 使用默认日志数据面标签，请在正式环境通过 .env 固定不可变版本"
  fi
  if ! grep -q 'OPSHUB_LOG_EXPORT_MAX_ATTEMPTS' "${rendered_file}" || ! grep -q 'OPSHUB_LOG_EXPORT_LEASE_SECONDS' "${rendered_file}"; then
    fail_check "Compose 缺少日志导出重试或租约参数"
  fi
  if [[ "${strict_mode}" == true ]] && grep -qE 'OPSHUB_LOG_QUEUE_MODE:[[:space:]]+direct' "${rendered_file}"; then
    fail_check "生产 Compose 必须把 OPSHUB_LOG_QUEUE_MODE 设置为 kafka"
  fi
  if [[ "${strict_mode}" == true ]] && ! grep -qE '^  redpanda:$' "${rendered_file}"; then
    fail_check "生产 Compose 未渲染 Redpanda；请使用 standard profile 或配置外部 Kafka"
  fi
  if [[ "${strict_mode}" == true ]] && [[ $(grep -c 'host_ip: 127.0.0.1' "${rendered_file}" || true) -lt 3 ]]; then
    fail_check "生产 Compose 必须把 MySQL、Redis、ClickHouse 管理端口绑定到 127.0.0.1"
  fi
  if [[ "${strict_mode}" == true ]] && grep -qE 'change-in-production|REPLACE_WITH_|OpsHub@(2026|Redis|ClickHouse)|:(latest|dev)([[:space:]]|$)' "${rendered_file}"; then
    fail_check "严格模式发现 Compose 默认密钥或可变镜像标签"
  fi
	if ! grep -q 'OPSHUB_LOG_TTL_MERGE_TIMEOUT_SECONDS' "${rendered_file}"; then
		fail_check "Compose 缺少 ClickHouse TTL 清理调度参数"
	fi
  printf 'PASS: Docker Compose 配置有效\n'
}

check_repository_state() {
	require_command git
	if ! git -C "${repository_directory}" diff --quiet || ! git -C "${repository_directory}" diff --cached --quiet; then
		fail_check "严格模式要求已提交的干净工作区"
	fi
	if [[ -n $(git -C "${repository_directory}" ls-files --others --exclude-standard | head -n 1) ]]; then
		fail_check "严格模式发现未跟踪文件，请提交或加入 .gitignore"
	fi
	printf 'PASS: Git 工作区可复现\n'
}

check_helm_config() {
  require_command helm
  local rendered_file
  rendered_file=$(mktemp)
  temporary_files+=("${rendered_file}")
  local helm_arguments=("${release_name}" "${chart_directory}" --namespace "${namespace}")
  local helm_lint_arguments=("${chart_directory}")
  if [[ -n "${values_file}" ]]; then
    helm_arguments+=(--values "${values_file}")
    helm_lint_arguments+=(--values "${values_file}")
  fi
  helm lint "${helm_lint_arguments[@]}" >/dev/null
  helm template "${helm_arguments[@]}" >"${rendered_file}"

  if ! grep -qE 'kind: Deployment' "${rendered_file}"; then
    fail_check "Helm 渲染结果没有 Deployment"
  fi
  if ! grep -qE 'name: .*backend' "${rendered_file}" || ! grep -qE 'name: .*frontend' "${rendered_file}"; then
    fail_check "Helm 渲染结果缺少 backend/frontend"
  fi
  if ! grep -qE 'kind: PodDisruptionBudget' "${rendered_file}"; then
    warn_check "Helm 渲染结果没有 PDB；单副本或显式关闭 PDB 时可接受"
  fi
  if ! grep -q 'OPSHUB_LOG_EXPORT_MAX_ATTEMPTS' "${rendered_file}" || ! grep -q 'OPSHUB_LOG_EXPORT_LEASE_SECONDS' "${rendered_file}"; then
    fail_check "Helm 缺少日志导出重试或租约参数"
  fi
  if [[ "${strict_mode}" == true ]] && grep -A1 'name: OPSHUB_LOG_QUEUE_MODE' "${rendered_file}" | grep -q 'value: direct'; then
    fail_check "生产 Helm values 必须把 logCenter.ingest.queue.mode 设置为 kafka"
  fi
  if [[ "${strict_mode}" == true ]] && grep -qE 'change-in-production|please-change-in-production|REPLACE_WITH_|OpsHub@(2026|Redis|ClickHouse)|:(latest|dev)([[:space:]]|$)' "${rendered_file}"; then
    fail_check "严格模式发现 Helm 默认密钥或 latest 镜像标签"
  fi
  printf 'PASS: Helm lint 与模板渲染有效\n'
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose)
      check_compose=true
      shift
      ;;
    --helm)
      check_helm=true
      shift
      ;;
    --values)
      [[ $# -ge 2 ]] || fail_check "--values 缺少文件路径"
      values_file=$2
      shift 2
      ;;
    --release)
      [[ $# -ge 2 ]] || fail_check "--release 缺少名称"
      release_name=$2
      shift 2
      ;;
    --namespace)
      [[ $# -ge 2 ]] || fail_check "--namespace 缺少名称"
      namespace=$2
      shift 2
      ;;
    --strict)
      strict_mode=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail_check "未知参数: $1"
      ;;
  esac
done

if [[ "${check_compose}" == false && "${check_helm}" == false ]]; then
  check_compose=true
  check_helm=true
fi

if [[ -n "${values_file}" && ! -f "${values_file}" ]]; then
  fail_check "Helm values 文件不存在: ${values_file}"
fi

if [[ "${check_compose}" == true ]]; then
  check_compose_config
fi
if [[ "${check_helm}" == true ]]; then
  check_helm_config
fi
if [[ "${strict_mode}" == true ]]; then
	check_repository_state
fi

printf '发布前检查完成：未执行部署或重启操作。\n'
