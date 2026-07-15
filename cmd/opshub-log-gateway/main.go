package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ydcloud-dy/opshub/internal/loggingest"
	"google.golang.org/grpc"
)

func main() {
	address := envString("OPSHUB_LOG_GATEWAY_HTTP_ADDRESS", ":9880")
	queueMode := strings.ToLower(envString("OPSHUB_LOG_QUEUE_MODE", "direct"))
	agentTokens := splitList(os.Getenv("OPSHUB_LOG_INGEST_TOKENS"))
	if len(agentTokens) == 0 {
		log.Fatal("OPSHUB_LOG_INGEST_TOKENS 不能为空")
	}
	writerToken := strings.TrimSpace(os.Getenv("OPSHUB_LOG_WRITER_TOKEN"))
	if queueMode == "direct" && writerToken == "" {
		log.Fatal("OPSHUB_LOG_WRITER_TOKEN 不能为空")
	}
	var publisher loggingest.BatchPublisher
	var kafkaConfig loggingest.KafkaConfig
	if queueMode == "kafka" {
		kafkaConfig = kafkaConfigFromEnv("gateway")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		var err error
		publisher, err = loggingest.NewKafkaPublisher(ctx, kafkaConfig)
		cancel()
		if err != nil {
			log.Fatalf("初始化 Kafka Publisher 失败: %v", err)
		}
	} else if queueMode != "direct" {
		log.Fatalf("不支持的日志队列模式: %s", queueMode)
	}

	gateway := loggingest.NewGateway(loggingest.GatewayConfig{
		WriterURL:           envString("OPSHUB_LOG_WRITER_URL", "http://127.0.0.1:9881"),
		WriterToken:         writerToken,
		QueueMode:           queueMode,
		QueueTopic:          kafkaConfig.Topic,
		BrokerCount:         len(kafkaConfig.Brokers),
		Publisher:           publisher,
		AgentTokens:         agentTokens,
		MaxBodyBytes:        int64(envInt("OPSHUB_LOG_GATEWAY_MAX_BODY_MB", 4)) * 1024 * 1024,
		RequestTimeout:      time.Duration(envInt("OPSHUB_LOG_GATEWAY_WRITER_TIMEOUT_SECONDS", 30)) * time.Second,
		RatePerSecond:       envInt("OPSHUB_LOG_GATEWAY_RATE_PER_SECOND", 5000),
		BurstRecords:        envInt("OPSHUB_LOG_GATEWAY_BURST_RECORDS", 10000),
		GlobalRatePerSecond: envInt("OPSHUB_LOG_GATEWAY_GLOBAL_RATE_PER_SECOND", 50000),
		GlobalBurstRecords:  envInt("OPSHUB_LOG_GATEWAY_GLOBAL_BURST_RECORDS", 100000),
		MaxInflight:         envInt("OPSHUB_LOG_GATEWAY_MAX_INFLIGHT", 256),
		Limits:              loggingest.DefaultLimits(),
		HTTPAddress:         address,
		GRPCAddress:         envString("OPSHUB_LOG_GATEWAY_GRPC_ADDRESS", ":9882"),
	})
	server := &http.Server{
		Addr:              address,
		Handler:           gateway.HTTPHandler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	grpcAddress := envString("OPSHUB_LOG_GATEWAY_GRPC_ADDRESS", ":9882")
	grpcListener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		log.Fatalf("Log Gateway gRPC 监听失败: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(envInt("OPSHUB_LOG_GATEWAY_MAX_BODY_MB", 4)*1024*1024),
		grpc.MaxSendMsgSize(1024*1024),
	)
	loggingest.RegisterGRPCGateway(grpcServer, gateway)
	go func() {
		log.Printf("OpsHub Log Gateway listening on %s", address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Log Gateway 启动失败: %v", err)
		}
	}()
	go func() {
		log.Printf("OpsHub Log Gateway gRPC listening on %s", grpcAddress)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("Log Gateway gRPC 服务停止: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()
	select {
	case <-grpcStopped:
	case <-time.After(10 * time.Second):
		grpcServer.Stop()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Log Gateway 停止失败: %v", err)
	}
	gateway.Close()
}

func kafkaConfigFromEnv(suffix string) loggingest.KafkaConfig {
	return loggingest.KafkaConfig{
		Brokers:           splitList(os.Getenv("OPSHUB_LOG_KAFKA_BROKERS")),
		Topic:             envString("OPSHUB_LOG_KAFKA_TOPIC", "opshub.logs.v1"),
		DeadletterTopic:   envString("OPSHUB_LOG_KAFKA_DEADLETTER_TOPIC", "opshub.logs.deadletter.v1"),
		ConsumerGroup:     envString("OPSHUB_LOG_KAFKA_CONSUMER_GROUP", "opshub-log-writer"),
		ClientID:          envString("OPSHUB_LOG_KAFKA_CLIENT_ID", "opshub-"+suffix),
		Partitions:        envInt("OPSHUB_LOG_KAFKA_PARTITIONS", 12),
		ReplicationFactor: envInt("OPSHUB_LOG_KAFKA_REPLICATION_FACTOR", 1),
		AutoCreateTopics:  envBool("OPSHUB_LOG_KAFKA_AUTO_CREATE_TOPICS", true),
		RetentionHours:    envInt("OPSHUB_LOG_KAFKA_RETENTION_HOURS", 72),
		TLS:               envBool("OPSHUB_LOG_KAFKA_TLS", false),
		TLSSkipVerify:     envBool("OPSHUB_LOG_KAFKA_TLS_SKIP_VERIFY", false),
		SASLMechanism:     strings.TrimSpace(os.Getenv("OPSHUB_LOG_KAFKA_SASL_MECHANISM")),
		Username:          strings.TrimSpace(os.Getenv("OPSHUB_LOG_KAFKA_USERNAME")),
		Password:          os.Getenv("OPSHUB_LOG_KAFKA_PASSWORD"),
		MaxPollRecords:    envInt("OPSHUB_LOG_KAFKA_MAX_POLL_RECORDS", 500),
		FetchMaxBytes:     int32(envInt("OPSHUB_LOG_KAFKA_FETCH_MAX_MB", 64) * 1024 * 1024),
	}
}

func splitList(value string) []string {
	parts := strings.FieldsFunc(value, func(char rune) bool {
		return char == ',' || char == ';' || char == '\n'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
