# OpsHub Agent Gateway 部署说明

OpsHub Agent 默认是主动上报模式：Agent 采集主机信息后，请求 `serverUrl` 指向的 `/api/v1/public/agents/*` 接口。如果 OpsHub 部署在内网，而云上主机无法直接访问内网 OpsHub，就需要单独部署 `opshub-agent-gateway`。

## 推荐拓扑

```text
云上/公网主机 Agent
        |
        | HTTPS
        v
公网 Agent Gateway
        |
        | VPN / 专线 / frp / 内网穿透 / 安全组放通
        v
内网 OpsHub
```

结论：Agent Gateway 需要单独部署在云上主机可以访问的位置，同时 Gateway 自己必须能访问内网 OpsHub。

## Gateway 做了什么

- 只转发 Agent 需要的公开接口，不暴露完整 OpsHub 后端。
- 支持健康检查：`/healthz`、`/readyz`。
- 支持限制来源 CIDR，减少公网暴露面。
- 支持请求体大小限制，避免异常大包打到内网 OpsHub。
- 这是轻量转发模式，不是消息队列；如果 Gateway 到 OpsHub 的链路中断，Agent 会在下次采集周期继续尝试上报。

允许转发的接口：

```text
GET  /api/v1/public/agents/install.sh
GET  /api/v1/public/agents/binaries/*
POST /api/v1/public/agents/register
POST /api/v1/public/agents/heartbeat
POST /api/v1/public/agents/metrics
```

## 编译

```bash
make agent-gateway
make agent-gateway-binaries
```

生成文件：

```text
bin/opshub-agent-gateway
bin/opshub-agent-gateway-linux-amd64
bin/opshub-agent-gateway-linux-arm64
```

## 启动示例

公网机器上启动 Gateway，`OPSHUB_UPSTREAM_URL` 指向内网 OpsHub 地址：

```bash
OPSHUB_UPSTREAM_URL=http://10.0.0.10:9876 \
./bin/opshub-agent-gateway --listen :9877
```

如果希望 Gateway 自己直接提供 HTTPS：

```bash
OPSHUB_UPSTREAM_URL=http://10.0.0.10:9876 \
./bin/opshub-agent-gateway \
  --listen :9877 \
  --tls-cert /etc/opshub-gateway/fullchain.pem \
  --tls-key /etc/opshub-gateway/privkey.pem
```

也可以用 Nginx/Caddy/SLB 在前面终止 HTTPS，再转发到本机 `127.0.0.1:9877`。

## 常用参数

```text
--listen              监听地址，默认 :9877
--upstream            内网 OpsHub 地址，也可用 OPSHUB_UPSTREAM_URL
--tls-cert            TLS 证书文件
--tls-key             TLS 私钥文件
--allow-cidrs         可选，允许访问的来源 CIDR，多个用逗号分隔
--max-body-mb         最大 POST 请求体，默认 10 MiB
```

环境变量同名：

```text
OPSHUB_GATEWAY_LISTEN
OPSHUB_UPSTREAM_URL
OPSHUB_GATEWAY_TLS_CERT
OPSHUB_GATEWAY_TLS_KEY
OPSHUB_GATEWAY_ALLOW_CIDRS
OPSHUB_GATEWAY_MAX_BODY_MB
```

## 前端安装时怎么填

进入 `资产管理 -> Agent管理`，把 `Agent访问地址` 填成云上主机可访问的 Gateway 地址，例如：

```text
https://agent-gateway.example.com
```

这样生成的安装命令会把 Agent 的 `serverUrl` 写成 Gateway 地址，Agent 后续注册、心跳、指标都会先发到 Gateway，再由 Gateway 转发到内网 OpsHub。

## systemd 示例

```ini
[Unit]
Description=OpsHub Agent Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=OPSHUB_UPSTREAM_URL=http://10.0.0.10:9876
ExecStart=/usr/local/bin/opshub-agent-gateway --listen :9877
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## 安全建议

- 公网入口建议使用 HTTPS。
- 只开放 Gateway 端口，不要直接把完整 OpsHub 后端暴露到公网。
- 如果云上 Agent 的出口 IP 固定，可以配置 `--allow-cidrs`。
- Gateway 到内网 OpsHub 的访问建议走 VPN、专线、frp 或其他受控链路。
