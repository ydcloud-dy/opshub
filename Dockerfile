
# Build stage
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/golang:1.25-alpine AS builder
# 设置 Go 环境变量
ENV GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux
# Install build dependencies
# RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy source code
COPY . .

# Copy go mod files
#COPY go.mod go.sum ./
# Download dependencies
RUN go mod download
# Build the application and Linux Agent binaries used by one-click install
RUN CGO_ENABLED=0 GOOS=linux go build -o opshub main.go && \
    mkdir -p data/agent-binaries && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=0.1.0" -o data/agent-binaries/opshub-agent-linux-amd64 ./cmd/opshub-agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=0.1.0" -o data/agent-binaries/opshub-agent-linux-arm64 ./cmd/opshub-agent

# Runtime stage
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/selectdb/alpine:latest

# Install ca-certificates, tzdata and kubectl
RUN apk --no-cache add ca-certificates tzdata curl && \
    curl -LO "https://mirrors.aliyun.com/kubernetes/kubectl/v1.29.0/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

# Set timezone
ENV TZ=Asia/Shanghai

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/opshub .
COPY --from=builder /build/data/agent-binaries ./data/agent-binaries
COPY data/GeoLite2-City.mmdb ./data/GeoLite2-City.mmdb

# Copy config template as default config
COPY config/config.yaml.example config/config.yaml

# Create runtime directories. Keep /app/data available for bundled Agent binaries.
RUN mkdir -p logs data/terminal-recordings && chmod 755 data/agent-binaries/opshub-agent-linux-*

# Expose port
EXPOSE 9876

# Run the application
CMD ["./opshub", "server"]
