package asset

import "strings"

func renderAgentInstallScript(serverURL, enrollmentToken, interval string) string {
	replacer := strings.NewReplacer(
		"{{SERVER_URL}}", shellSingleQuote(strings.TrimRight(serverURL, "/")),
		"{{ENROLLMENT_TOKEN}}", shellSingleQuote(enrollmentToken),
		"{{INTERVAL}}", shellSingleQuote(interval),
	)
	return replacer.Replace(agentInstallScriptTemplate)
}

func shellSingleQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

const agentInstallScriptTemplate = `#!/usr/bin/env bash
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 执行，或通过 sudo bash 安装 OpsHub Agent" >&2
  exit 1
fi

SERVER_URL={{SERVER_URL}}
ENROLLMENT_TOKEN={{ENROLLMENT_TOKEN}}
INTERVAL={{INTERVAL}}
CONFIG_DIR="/etc/opshub-agent"
CONFIG_FILE="${CONFIG_DIR}/agent.json"
BIN_PATH="/usr/local/bin/opshub-agent"
START_SCRIPT="/usr/local/bin/opshub-agent-start"
SERVICE_FILE="/etc/systemd/system/opshub-agent.service"
LOG_FILE="/var/log/opshub-agent.log"

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo "不支持的CPU架构: $(uname -m)" >&2; exit 1 ;;
  esac
}

ARCH="$(detect_arch)"
BINARY_URL="${SERVER_URL}/api/v1/public/agents/binaries/opshub-agent-linux-${ARCH}"

echo "下载 OpsHub Agent: ${BINARY_URL}"
if ! curl -fsSL "${BINARY_URL}" -o "${BIN_PATH}.tmp"; then
  echo "下载 Agent 二进制失败。请检查 Agent访问地址是否能访问 OpsHub 或 Agent Gateway，并确认 /api/v1/public/agents/binaries/opshub-agent-linux-${ARCH} 可下载" >&2
  exit 1
fi
chmod 755 "${BIN_PATH}.tmp"
mv "${BIN_PATH}.tmp" "${BIN_PATH}"

mkdir -p "${CONFIG_DIR}"
chmod 700 "${CONFIG_DIR}"
cat > "${CONFIG_FILE}" <<EOF
{
  "serverUrl": "${SERVER_URL}",
  "enrollmentToken": "${ENROLLMENT_TOKEN}",
  "interval": ${INTERVAL}
}
EOF
chmod 600 "${CONFIG_FILE}"

cat > "${START_SCRIPT}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
if command -v pkill >/dev/null 2>&1; then
  pkill -f "${BIN_PATH} --config ${CONFIG_FILE}" || true
fi
nohup "${BIN_PATH}" --config "${CONFIG_FILE}" >>"${LOG_FILE}" 2>&1 &
echo \$! > /var/run/opshub-agent.pid
EOF
chmod 755 "${START_SCRIPT}"

systemd_available() {
  command -v systemctl >/dev/null 2>&1 || return 1
  [ -d /run/systemd/system ] || return 1
  systemctl list-units >/dev/null 2>&1 || return 1
}

install_cron_reboot() {
  command -v crontab >/dev/null 2>&1 || return 1
  local cron_line="@reboot ${START_SCRIPT} >/dev/null 2>&1"
  local current_cron
  current_cron="$(crontab -l 2>/dev/null || true)"
  if printf '%s\n' "${current_cron}" | grep -Fq "${START_SCRIPT}"; then
    return 0
  fi
  {
    printf '%s\n' "${current_cron}"
    printf '%s\n' "${cron_line}"
  } | sed '/^$/d' | crontab -
}

install_systemd() {
  cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=OpsHub Host Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_PATH} --config ${CONFIG_FILE}
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload || return 1
  if systemctl is-active --quiet opshub-agent; then
    systemctl restart opshub-agent || return 1
  else
    systemctl enable --now opshub-agent || return 1
  fi
}

start_direct_agent() {
  "${START_SCRIPT}"
  if install_cron_reboot; then
    echo "OpsHub Agent 已在非 systemd 模式下启动，并已写入 crontab @reboot 自启"
  else
    echo "OpsHub Agent 已在非 systemd 模式下后台启动；当前系统没有可用 crontab，重启后需要再次执行 ${START_SCRIPT}"
  fi
}

if systemd_available; then
  if install_systemd; then
    echo "OpsHub Agent 已安装并启动: systemctl status opshub-agent"
  else
    echo "systemd 服务安装或启动失败，自动降级为后台进程模式" >&2
    start_direct_agent
  fi
else
  start_direct_agent
fi
`
