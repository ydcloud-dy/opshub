package loggingest

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type KafkaConfig struct {
	Brokers           []string
	Topic             string
	DeadletterTopic   string
	ConsumerGroup     string
	ClientID          string
	Partitions        int
	ReplicationFactor int
	AutoCreateTopics  bool
	RetentionHours    int
	TLS               bool
	TLSSkipVerify     bool
	SASLMechanism     string
	Username          string
	Password          string
	MaxPollRecords    int
	FetchMaxBytes     int32
}

func (config *KafkaConfig) normalize() {
	cleanedBrokers := make([]string, 0, len(config.Brokers))
	for _, broker := range config.Brokers {
		if broker = strings.TrimSpace(broker); broker != "" {
			cleanedBrokers = append(cleanedBrokers, broker)
		}
	}
	config.Brokers = cleanedBrokers
	if strings.TrimSpace(config.Topic) == "" {
		config.Topic = "opshub.logs.v1"
	}
	if strings.TrimSpace(config.DeadletterTopic) == "" {
		config.DeadletterTopic = "opshub.logs.deadletter.v1"
	}
	if strings.TrimSpace(config.ConsumerGroup) == "" {
		config.ConsumerGroup = "opshub-log-writer"
	}
	if strings.TrimSpace(config.ClientID) == "" {
		config.ClientID = "opshub-logcenter"
	}
	if config.Partitions <= 0 {
		config.Partitions = 12
	}
	if config.ReplicationFactor <= 0 {
		config.ReplicationFactor = 1
	}
	if config.RetentionHours <= 0 {
		config.RetentionHours = 72
	}
	if config.MaxPollRecords <= 0 {
		config.MaxPollRecords = 500
	}
	if config.FetchMaxBytes <= 0 {
		config.FetchMaxBytes = 64 * 1024 * 1024
	}
	config.SASLMechanism = strings.ToLower(strings.TrimSpace(config.SASLMechanism))
}

func (config KafkaConfig) validate() error {
	if len(config.Brokers) == 0 {
		return fmt.Errorf("Kafka brokers 不能为空")
	}
	if strings.TrimSpace(config.Topic) == "" || strings.TrimSpace(config.DeadletterTopic) == "" {
		return fmt.Errorf("Kafka 日志 Topic 和死信 Topic 不能为空")
	}
	if config.SASLMechanism != "" && (strings.TrimSpace(config.Username) == "" || config.Password == "") {
		return fmt.Errorf("Kafka SASL 已启用但用户名或密码为空")
	}
	switch config.SASLMechanism {
	case "", "plain", "scram-sha-256", "scram-sha-512":
		return nil
	default:
		return fmt.Errorf("不支持的 Kafka SASL 机制: %s", config.SASLMechanism)
	}
}

func kafkaCommonOptions(config KafkaConfig, clientSuffix string) ([]kgo.Opt, error) {
	config.normalize()
	if err := config.validate(); err != nil {
		return nil, err
	}
	options := []kgo.Opt{
		kgo.SeedBrokers(config.Brokers...),
		kgo.ClientID(strings.TrimSpace(config.ClientID) + "-" + clientSuffix),
		kgo.DialTimeout(10 * time.Second),
		kgo.RequestTimeoutOverhead(10 * time.Second),
	}
	if config.TLS {
		options = append(options, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: config.TLSSkipVerify})) //nolint:gosec
	}
	switch config.SASLMechanism {
	case "plain":
		options = append(options, kgo.SASL(plain.Auth{User: config.Username, Pass: config.Password}.AsMechanism()))
	case "scram-sha-256":
		options = append(options, kgo.SASL(scram.Auth{User: config.Username, Pass: config.Password}.AsSha256Mechanism()))
	case "scram-sha-512":
		options = append(options, kgo.SASL(scram.Auth{User: config.Username, Pass: config.Password}.AsSha512Mechanism()))
	}
	return options, nil
}

func EnsureKafkaTopics(ctx context.Context, config KafkaConfig) error {
	config.normalize()
	if !config.AutoCreateTopics {
		return nil
	}
	options, err := kafkaCommonOptions(config, "admin")
	if err != nil {
		return err
	}
	client, err := kgo.NewClient(options...)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("连接 Kafka 失败: %w", err)
	}
	retentionMS := fmt.Sprintf("%d", time.Duration(config.RetentionHours)*time.Hour/time.Millisecond)
	configs := map[string]*string{"retention.ms": &retentionMS, "cleanup.policy": kadm.StringPtr("delete")}
	responses, err := kadm.NewClient(client).CreateTopics(
		ctx, int32(config.Partitions), int16(config.ReplicationFactor), configs,
		config.Topic, config.DeadletterTopic,
	)
	if err != nil {
		return fmt.Errorf("创建 Kafka Topic 失败: %w", err)
	}
	for _, response := range responses {
		if response.Err != nil && !errors.Is(response.Err, kerr.TopicAlreadyExists) {
			return fmt.Errorf("创建 Kafka Topic %s 失败: %w", response.Topic, response.Err)
		}
	}
	return nil
}

type KafkaPublisher struct {
	config KafkaConfig
	client *kgo.Client
	once   sync.Once
}

func NewKafkaPublisher(ctx context.Context, config KafkaConfig) (*KafkaPublisher, error) {
	config.normalize()
	if err := EnsureKafkaTopics(ctx, config); err != nil {
		return nil, err
	}
	options, err := kafkaCommonOptions(config, "gateway")
	if err != nil {
		return nil, err
	}
	options = append(options,
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.ProducerBatchCompression(kgo.ZstdCompression(), kgo.SnappyCompression()),
		kgo.ProducerLinger(5*time.Millisecond),
		kgo.ProducerBatchMaxBytes(4*1024*1024),
		kgo.MaxBufferedRecords(10000),
		kgo.MaxBufferedBytes(128*1024*1024),
	)
	if config.AutoCreateTopics {
		options = append(options, kgo.AllowAutoTopicCreation())
	}
	client, err := kgo.NewClient(options...)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("连接 Kafka 失败: %w", err)
	}
	return &KafkaPublisher{config: config, client: client}, nil
}

func (publisher *KafkaPublisher) Publish(ctx context.Context, batch LogBatch) (IngestAck, error) {
	raw, err := json.Marshal(batch)
	if err != nil {
		return IngestAck{}, err
	}
	record := &kgo.Record{
		Topic: publisher.config.Topic,
		Key:   []byte(firstNonEmptyKafka(batch.AgentID, fmt.Sprintf("%s:%d", batch.AssetType, batch.AssetID))),
		Value: raw,
		Headers: []kgo.RecordHeader{
			{Key: "opshub-batch-id", Value: []byte(batch.BatchID)},
			{Key: "opshub-agent-id", Value: []byte(batch.AgentID)},
		},
	}
	if err := publisher.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return IngestAck{}, fmt.Errorf("写入 Kafka 失败: %w", err)
	}
	return successAck(batch, false), nil
}

func (publisher *KafkaPublisher) Close() {
	publisher.once.Do(func() { publisher.client.Close() })
}

type KafkaConsumer struct {
	config KafkaConfig
	client *kgo.Client
	writer *Writer
	once   sync.Once
}

func NewKafkaConsumer(ctx context.Context, config KafkaConfig, writer *Writer) (*KafkaConsumer, error) {
	if writer == nil {
		return nil, fmt.Errorf("Kafka Consumer 缺少 Writer")
	}
	config.normalize()
	if err := EnsureKafkaTopics(ctx, config); err != nil {
		return nil, err
	}
	options, err := kafkaCommonOptions(config, "writer")
	if err != nil {
		return nil, err
	}
	options = append(options,
		kgo.ConsumeTopics(config.Topic),
		kgo.ConsumerGroup(config.ConsumerGroup),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
		kgo.Balancers(kgo.CooperativeStickyBalancer()),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.FetchMaxBytes(config.FetchMaxBytes),
		kgo.FetchMaxPartitionBytes(8*1024*1024),
	)
	client, err := kgo.NewClient(options...)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("连接 Kafka 失败: %w", err)
	}
	writer.SetQueueState(true, 0, nil)
	return &KafkaConsumer{config: config, client: client, writer: writer}, nil
}

type kafkaPartitionResult struct {
	commit *kgo.Record
	lag    int64
	err    error
}

func (consumer *KafkaConsumer) Run(ctx context.Context) error {
	for {
		fetches := consumer.client.PollRecords(ctx, consumer.config.MaxPollRecords)
		if ctx.Err() != nil {
			consumer.client.AllowRebalance()
			return nil
		}
		fetchErrors := fetches.Errors()
		if len(fetchErrors) > 0 {
			fetchErr := fetchErrors[0].Err
			consumer.writer.SetQueueState(false, consumer.writer.queueLag.Load(), fetchErr)
			if fetches.NumRecords() == 0 {
				continue
			}
		}
		results := make(chan kafkaPartitionResult)
		partitionCount := 0
		fetches.EachPartition(func(partition kgo.FetchTopicPartition) {
			if len(partition.Records) == 0 {
				return
			}
			partitionCount++
			go func() { results <- consumer.processPartition(ctx, partition) }()
		})
		commits := make([]*kgo.Record, 0, partitionCount)
		var remainingLag int64
		var processErr error
		for index := 0; index < partitionCount; index++ {
			result := <-results
			remainingLag += result.lag
			if result.commit != nil {
				commits = append(commits, result.commit)
			}
			if result.err != nil && processErr == nil {
				processErr = result.err
			}
		}
		if processErr == nil && len(commits) > 0 {
			processErr = consumer.commitWithRetry(ctx, commits)
		}
		consumer.client.AllowRebalance()
		if ctx.Err() != nil {
			return nil
		}
		consumer.writer.SetQueueState(processErr == nil, remainingLag, processErr)
	}
}

func (consumer *KafkaConsumer) processPartition(ctx context.Context, partition kgo.FetchTopicPartition) kafkaPartitionResult {
	if partition.Err != nil {
		return kafkaPartitionResult{err: partition.Err}
	}
	var last *kgo.Record
	for _, record := range partition.Records {
		if err := consumer.processRecord(ctx, record); err != nil {
			return kafkaPartitionResult{commit: last, err: err}
		}
		last = record
	}
	lag := int64(0)
	if last != nil && partition.HighWatermark > last.Offset+1 {
		lag = partition.HighWatermark - last.Offset - 1
	}
	return kafkaPartitionResult{commit: last, lag: lag}
}

func (consumer *KafkaConsumer) processRecord(ctx context.Context, record *kgo.Record) error {
	var batch LogBatch
	if err := json.Unmarshal(record.Value, &batch); err != nil {
		return consumer.publishDeadletterWithRetry(ctx, record, "INVALID_JSON", err)
	}
	if err := ValidateBatch(batch, DefaultLimits()); err != nil {
		return consumer.publishDeadletterWithRetry(ctx, record, "INVALID_BATCH", err)
	}
	backoff := 250 * time.Millisecond
	for {
		ack := consumer.writer.Submit(ctx, batch)
		if ack.ErrorCode == "" {
			return nil
		}
		writeErr := fmt.Errorf("%s: %s", ack.ErrorCode, ack.ErrorMessage)
		consumer.writer.SetQueueState(false, consumer.writer.queueLag.Load(), writeErr)
		if err := waitKafkaRetry(ctx, backoff); err != nil {
			return err
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (consumer *KafkaConsumer) publishDeadletterWithRetry(ctx context.Context, record *kgo.Record, code string, cause error) error {
	payload, err := json.Marshal(map[string]interface{}{
		"failedAt": time.Now().UTC(), "errorCode": code, "error": cause.Error(),
		"sourceTopic": record.Topic, "sourcePartition": record.Partition, "sourceOffset": record.Offset,
		"key": string(record.Key), "payloadBase64": record.Value,
	})
	if err != nil {
		return err
	}
	backoff := 250 * time.Millisecond
	for {
		deadletter := &kgo.Record{Topic: consumer.config.DeadletterTopic, Key: record.Key, Value: payload}
		if publishErr := consumer.client.ProduceSync(ctx, deadletter).FirstErr(); publishErr == nil {
			consumer.writer.MarkDeadletter()
			return nil
		} else if err := waitKafkaRetry(ctx, backoff); err != nil {
			return fmt.Errorf("写入 Kafka 死信 Topic 失败: %w", publishErr)
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (consumer *KafkaConsumer) commitWithRetry(ctx context.Context, records []*kgo.Record) error {
	backoff := 250 * time.Millisecond
	for {
		if err := consumer.client.CommitRecords(ctx, records...); err == nil {
			return nil
		} else if waitErr := waitKafkaRetry(ctx, backoff); waitErr != nil {
			return err
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (consumer *KafkaConsumer) Close() {
	consumer.once.Do(func() { consumer.client.CloseAllowingRebalance() })
}

func waitKafkaRetry(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func firstNonEmptyKafka(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "unknown"
}
