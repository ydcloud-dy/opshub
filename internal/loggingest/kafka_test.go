package loggingest

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func testKafkaConfig(cluster *kfake.Cluster, group string) KafkaConfig {
	return KafkaConfig{
		Brokers: cluster.ListenAddrs(), Topic: "opshub.logs.v1", DeadletterTopic: "opshub.logs.deadletter.v1",
		ConsumerGroup: group, ClientID: "opshub-test", Partitions: 6, ReplicationFactor: 1,
		MaxPollRecords: 100, FetchMaxBytes: 8 * 1024 * 1024,
	}
}

func newFakeKafkaCluster(t *testing.T) *kfake.Cluster {
	t.Helper()
	cluster, err := kfake.NewCluster(
		kfake.NumBrokers(3),
		kfake.SeedTopics(6, "opshub.logs.v1"),
		kfake.SeedTopics(3, "opshub.logs.deadletter.v1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cluster.Close)
	return cluster
}

func TestKafkaGatewayWriterRoundTrip(t *testing.T) {
	cluster := newFakeKafkaCluster(t)
	config := testKafkaConfig(cluster, "roundtrip")
	sink := &testSink{}
	sink.failures.Store(-1)
	writer := NewWriter(WriterConfig{Workers: 2, QueueMode: "kafka", QueueTopic: config.Topic, ConsumerGroup: config.ConsumerGroup}, sink)
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	consumer, err := NewKafkaConsumer(ctx, config, writer)
	if err != nil {
		t.Fatal(err)
	}
	consumerDone := make(chan error, 1)
	go func() { consumerDone <- consumer.Run(ctx) }()
	defer func() {
		cancel()
		consumer.Close()
		<-consumerDone
	}()

	publisher, err := NewKafkaPublisher(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	gateway := NewGateway(GatewayConfig{
		QueueMode: "kafka", QueueTopic: config.Topic, BrokerCount: len(config.Brokers), Publisher: publisher,
		AgentTokens: []string{"agent-token"}, GlobalRatePerSecond: 100000, GlobalBurstRecords: 100000,
	})
	defer gateway.Close()
	if ack := gateway.Submit(context.Background(), "agent-token", validBatch()); ack.ErrorCode != "" {
		t.Fatalf("Kafka publish failed: %+v", ack)
	}
	waitForKafkaCondition(t, 10*time.Second, func() bool { return sink.calls.Load() == 1 })
	if status := writer.Status(); status.AcceptedRecords != 1 || !status.QueueHealthy {
		t.Fatalf("unexpected writer status: %+v", status)
	}
}

func TestKafkaConsumerRetriesUntilStorageRecovers(t *testing.T) {
	cluster := newFakeKafkaCluster(t)
	config := testKafkaConfig(cluster, "storage-recovery")
	sink := &testSink{}
	sink.failures.Store(2)
	writer := NewWriter(WriterConfig{Workers: 1, MaxRetries: 0, QueueMode: "kafka", QueueTopic: config.Topic, ConsumerGroup: config.ConsumerGroup}, sink)
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	consumer, err := NewKafkaConsumer(ctx, config, writer)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	defer func() {
		cancel()
		consumer.Close()
		<-done
	}()
	publisher, err := NewKafkaPublisher(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	if _, err := publisher.Publish(context.Background(), validBatch()); err != nil {
		t.Fatal(err)
	}
	waitForKafkaCondition(t, 10*time.Second, func() bool { return writer.Status().AcceptedRecords == 1 })
	if sink.calls.Load() < 3 {
		t.Fatalf("expected storage retries, calls = %d", sink.calls.Load())
	}
	if writer.Status().DeadletterBatches != 0 {
		t.Fatal("transient storage failure must not enter deadletter in Kafka mode")
	}
}

func TestKafkaConsumerMovesInvalidPayloadToDeadletter(t *testing.T) {
	cluster := newFakeKafkaCluster(t)
	config := testKafkaConfig(cluster, "deadletter")
	sink := &testSink{}
	sink.failures.Store(-1)
	writer := NewWriter(WriterConfig{Workers: 1, QueueMode: "kafka", QueueTopic: config.Topic, ConsumerGroup: config.ConsumerGroup}, sink)
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	consumer, err := NewKafkaConsumer(ctx, config, writer)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	defer func() {
		cancel()
		consumer.Close()
		<-done
	}()
	client, err := kgo.NewClient(kgo.SeedBrokers(config.Brokers...))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.ProduceSync(context.Background(), &kgo.Record{Topic: config.Topic, Key: []byte("bad"), Value: []byte(`{"broken"`)}).FirstErr(); err != nil {
		t.Fatal(err)
	}
	waitForKafkaCondition(t, 10*time.Second, func() bool { return writer.Status().DeadletterBatches == 1 })
	if sink.calls.Load() != 0 {
		t.Fatalf("invalid payload reached ClickHouse sink: %d", sink.calls.Load())
	}
}

func TestKafkaConsumerGroupScalesAcrossWriters(t *testing.T) {
	cluster := newFakeKafkaCluster(t)
	config := testKafkaConfig(cluster, "horizontal-scale")
	sink := &testSink{}
	sink.failures.Store(-1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type runningConsumer struct {
		writer   *Writer
		consumer *KafkaConsumer
		done     chan error
	}
	running := make([]runningConsumer, 0, 2)
	for index := 0; index < 2; index++ {
		writer := NewWriter(WriterConfig{Workers: 2, QueueMode: "kafka", QueueTopic: config.Topic, ConsumerGroup: config.ConsumerGroup}, sink)
		consumer, err := NewKafkaConsumer(ctx, config, writer)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- consumer.Run(ctx) }()
		running = append(running, runningConsumer{writer: writer, consumer: consumer, done: done})
	}
	defer func() {
		cancel()
		for _, item := range running {
			item.consumer.Close()
			<-item.done
			item.writer.Close()
		}
	}()

	publisher, err := NewKafkaPublisher(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	for index := 1; index <= 100; index++ {
		batch := validBatch()
		batch.BatchID = fmt.Sprintf("00000000-0000-4000-8000-%012d", index)
		batch.AgentID = fmt.Sprintf("agent-%d", index%12)
		batch.Records[0].Body = fmt.Sprintf("message-%d", index)
		if _, err := publisher.Publish(context.Background(), batch); err != nil {
			t.Fatal(err)
		}
	}
	waitForKafkaCondition(t, 15*time.Second, func() bool { return sink.calls.Load() == 100 })
	var accepted uint64
	for _, item := range running {
		accepted += item.writer.Status().AcceptedBatches
	}
	if accepted != 100 {
		t.Fatalf("consumer group accepted %d batches, want 100", accepted)
	}
}

func TestKafkaPublisherContinuesAfterBrokerRemoval(t *testing.T) {
	cluster := newFakeKafkaCluster(t)
	config := testKafkaConfig(cluster, "broker-failover")
	publisher, err := NewKafkaPublisher(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()

	failedBroker := cluster.LeaderFor(config.Topic, 0)
	if err := cluster.RemoveNode(failedBroker); err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 20; index++ {
		batch := validBatch()
		batch.BatchID = fmt.Sprintf("00000000-0000-4000-8001-%012d", index)
		batch.AgentID = fmt.Sprintf("agent-%d", index%6)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, publishErr := publisher.Publish(ctx, batch)
		cancel()
		if publishErr != nil {
			t.Fatalf("publish after broker removal failed at batch %d: %v", index, publishErr)
		}
	}
	if _, _, err := cluster.AddNode(failedBroker, 0); err != nil {
		t.Fatal(err)
	}
}

func TestKafkaWriterRestartConsumesUncommittedBatch(t *testing.T) {
	cluster := newFakeKafkaCluster(t)
	config := testKafkaConfig(cluster, "writer-restart")
	publisher, err := NewKafkaPublisher(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	if _, err := publisher.Publish(context.Background(), validBatch()); err != nil {
		t.Fatal(err)
	}

	failingSink := &testSink{}
	failingSink.failures.Store(100000)
	firstWriter := NewWriter(WriterConfig{Workers: 1, MaxRetries: 0, QueueMode: "kafka", QueueTopic: config.Topic, ConsumerGroup: config.ConsumerGroup}, failingSink)
	firstCtx, firstCancel := context.WithCancel(context.Background())
	firstConsumer, err := NewKafkaConsumer(firstCtx, config, firstWriter)
	if err != nil {
		firstWriter.Close()
		t.Fatal(err)
	}
	firstDone := make(chan error, 1)
	go func() { firstDone <- firstConsumer.Run(firstCtx) }()
	waitForKafkaCondition(t, 10*time.Second, func() bool { return failingSink.calls.Load() > 0 })
	firstCancel()
	firstConsumer.Close()
	<-firstDone
	firstWriter.Close()

	recoveredSink := &testSink{}
	recoveredSink.failures.Store(-1)
	secondWriter := NewWriter(WriterConfig{Workers: 1, QueueMode: "kafka", QueueTopic: config.Topic, ConsumerGroup: config.ConsumerGroup}, recoveredSink)
	defer secondWriter.Close()
	secondCtx, secondCancel := context.WithCancel(context.Background())
	secondConsumer, err := NewKafkaConsumer(secondCtx, config, secondWriter)
	if err != nil {
		secondCancel()
		t.Fatal(err)
	}
	secondDone := make(chan error, 1)
	go func() { secondDone <- secondConsumer.Run(secondCtx) }()
	defer func() {
		secondCancel()
		secondConsumer.Close()
		<-secondDone
	}()
	waitForKafkaCondition(t, 10*time.Second, func() bool { return secondWriter.Status().AcceptedRecords == 1 })
	if recoveredSink.calls.Load() != 1 {
		t.Fatalf("uncommitted batch was not replayed exactly once: %d", recoveredSink.calls.Load())
	}
}

type recordCountingSink struct {
	batches atomic.Int64
	records atomic.Int64
}

func (sink *recordCountingSink) WriteBatch(_ context.Context, batch LogBatch) error {
	sink.batches.Add(1)
	sink.records.Add(int64(len(batch.Records)))
	return nil
}

func TestKafkaBurstFiftyThousandRecordsWithoutLoss(t *testing.T) {
	cluster := newFakeKafkaCluster(t)
	config := testKafkaConfig(cluster, "burst-50k")
	config.MaxPollRecords = 200
	sink := &recordCountingSink{}
	ctx, cancel := context.WithCancel(context.Background())

	type runningConsumer struct {
		writer   *Writer
		consumer *KafkaConsumer
		done     chan error
	}
	running := make([]runningConsumer, 0, 2)
	for index := 0; index < 2; index++ {
		writer := NewWriter(WriterConfig{Workers: 4, QueueCapacity: 512, QueueMode: "kafka", QueueTopic: config.Topic, ConsumerGroup: config.ConsumerGroup}, sink)
		consumer, err := NewKafkaConsumer(ctx, config, writer)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- consumer.Run(ctx) }()
		running = append(running, runningConsumer{writer: writer, consumer: consumer, done: done})
	}
	defer func() {
		cancel()
		for _, item := range running {
			item.consumer.Close()
			<-item.done
			item.writer.Close()
		}
	}()

	publisher, err := NewKafkaPublisher(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	const batchCount = 100
	const recordsPerBatch = 500
	for batchIndex := 1; batchIndex <= batchCount; batchIndex++ {
		batch := validBatch()
		batch.BatchID = fmt.Sprintf("00000000-0000-4000-8002-%012d", batchIndex)
		batch.AgentID = fmt.Sprintf("agent-%d", batchIndex%24)
		batch.SequenceStart = uint64((batchIndex-1)*recordsPerBatch + 1)
		batch.SequenceEnd = uint64(batchIndex * recordsPerBatch)
		batch.Records = make([]LogRecord, 0, recordsPerBatch)
		for recordIndex := 0; recordIndex < recordsPerBatch; recordIndex++ {
			sequence := batch.SequenceStart + uint64(recordIndex)
			batch.Records = append(batch.Records, LogRecord{
				Sequence: sequence, TimestampUnixNano: time.Now().UnixNano(), Body: fmt.Sprintf("message-%d", sequence), SeverityText: "INFO",
			})
		}
		if _, err := publisher.Publish(context.Background(), batch); err != nil {
			t.Fatalf("publish batch %d failed: %v", batchIndex, err)
		}
	}
	waitForKafkaCondition(t, 30*time.Second, func() bool { return sink.records.Load() == batchCount*recordsPerBatch })
	if sink.batches.Load() != batchCount {
		t.Fatalf("written batches = %d, want %d", sink.batches.Load(), batchCount)
	}
	var accepted uint64
	for _, item := range running {
		accepted += item.writer.Status().AcceptedRecords
	}
	if accepted != batchCount*recordsPerBatch {
		t.Fatalf("accepted records = %d, want %d", accepted, batchCount*recordsPerBatch)
	}
}

func waitForKafkaCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("timed out waiting for Kafka condition")
}
