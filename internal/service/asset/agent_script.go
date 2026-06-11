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
SERVICE_FILE="/etc/systemd/system/opshub-agent.service"

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
  echo "下载 Agent 二进制失败。请先在 OpsHub 服务器上执行: make agent-binaries" >&2
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

if command -v systemctl >/dev/null 2>&1; then
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
  systemctl daemon-reload
  if systemctl is-active --quiet opshub-agent; then
    systemctl restart opshub-agent
  else
    systemctl enable --now opshub-agent
  fi
  echo "OpsHub Agent 已安装并启动: systemctl status opshub-agent"
else
  if command -v pkill >/dev/null 2>&1; then
    pkill -f "${BIN_PATH} --config ${CONFIG_FILE}" || true
  fi
  nohup "${BIN_PATH}" --config "${CONFIG_FILE}" >/var/log/opshub-agent.log 2>&1 &
  echo "OpsHub Agent 已安装并以后台进程启动"
fi
`
