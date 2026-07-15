package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ydcloud-dy/opshub/internal/loggingest"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
)

func main() {
	address := envString("OPSHUB_LOG_WRITER_HTTP_ADDRESS", ":9881")
	queueMode := strings.ToLower(envString("OPSHUB_LOG_QUEUE_MODE", "direct"))
	writerToken := strings.TrimSpace(os.Getenv("OPSHUB_LOG_WRITER_TOKEN"))
	if writerToken == "" {
		log.Fatal("OPSHUB_LOG_WRITER_TOKEN 不能为空")
	}
	cluster := logmodel.StorageCluster{
		Name:                 "OpsHub 内置日志存储",
		StorageType:          "clickhouse",
		Endpoints:            envString("OPSHUB_LOGCENTER_CLICKHOUSE_ENDPOINT", "http://127.0.0.1:8123"),
		DatabaseName:         envString("OPSHUB_LOGCENTER_CLICKHOUSE_DATABASE", "opshub_logs"),
		Username:             strings.TrimSpace(os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_USERNAME")),
		SkipTLSVerify:        envBool("OPSHUB_LOGCENTER_CLICKHOUSE_SKIP_TLS_VERIFY", false),
		Timeout:              envInt("OPSHUB_LOGCENTER_CLICKHOUSE_TIMEOUT", 60),
		DefaultRetentionDays: envInt("OPSHUB_LOGCENTER_RETENTION_DAYS", 30),
		Enabled:              true,
		IsPrimary:            true,
	}
	password := os.Getenv("OPSHUB_LOGCENTER_CLICKHOUSE_PASSWORD")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cluster.Timeout)*time.Second)
	clickhouse := logsvc.NewClickHouseService()
	if err := clickhouse.Initialize(ctx, cluster, password); err != nil {
		cancel()
		log.Fatalf("初始化 ClickHouse 日志表失败: %v", err)
	}
	cancel()

	kafkaConfig := kafkaConfigFromEnv("writer")
	writerService := loggingest.NewWriter(loggingest.WriterConfig{
		QueueCapacity:  envInt("OPSHUB_LOG_WRITER_QUEUE_CAPACITY", 2048),
		Workers:        envInt("OPSHUB_LOG_WRITER_WORKERS", 4),
		MaxRetries:     envIntAllowZero("OPSHUB_LOG_WRITER_MAX_RETRIES", 3),
		WriteTimeout:   time.Duration(envInt("OPSHUB_LOG_WRITER_WRITE_TIMEOUT_SECONDS", 60)) * time.Second,
		DeadletterPath: envString("OPSHUB_LOG_WRITER_DEADLETTER_PATH", "/app/data/log-deadletter/failed.ndjson"),
		DedupCapacity:  envInt("OPSHUB_LOG_WRITER_DEDUP_CAPACITY", 100000),
		QueueMode:      queueMode,
		QueueTopic:     kafkaConfig.Topic,
		ConsumerGroup:  kafkaConfig.ConsumerGroup,
		BrokerCount:    len(kafkaConfig.Brokers),
	}, loggingest.NewClickHouseSink(cluster, password))
	var kafkaConsumer *loggingest.KafkaConsumer
	var consumerCancel context.CancelFunc
	consumerErrors := make(chan error, 1)
	if queueMode == "kafka" {
		consumerCtx, cancelConsumer := context.WithCancel(context.Background())
		consumerCancel = cancelConsumer
		var err error
		kafkaConsumer, err = loggingest.NewKafkaConsumer(consumerCtx, kafkaConfig, writerService)
		if err != nil {
			cancelConsumer()
			writerService.Close()
			log.Fatalf("初始化 Kafka Consumer 失败: %v", err)
		}
		go func() { consumerErrors <- kafkaConsumer.Run(consumerCtx) }()
	} else if queueMode != "direct" {
		writerService.Close()
		log.Fatalf("不支持的日志队列模式: %s", queueMode)
	}
	server := &http.Server{
		Addr:              address,
		Handler:           loggingest.WriterHTTPHandler(writerService, writerToken, int64(envInt("OPSHUB_LOG_WRITER_MAX_BODY_MB", 8))*1024*1024),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Printf("OpsHub Log Writer listening on %s", address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Log Writer 启动失败: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stop:
	case err := <-consumerErrors:
		if err != nil {
			log.Printf("Kafka Consumer 已停止: %v", err)
		}
	}
	if consumerCancel != nil {
		consumerCancel()
	}
	if kafkaConsumer != nil {
		kafkaConsumer.Close()
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Log Writer HTTP 服务停止失败: %v", err)
	}
	writerService.Close()
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

func envIntAllowZero(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 0 {
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
