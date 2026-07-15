package loggingest

import (
	"context"
	"io"
	"strings"

	logingestv1 "github.com/ydcloud-dy/opshub/api/logingest/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type GRPCGatewayServer struct {
	logingestv1.UnimplementedLogIngestServiceServer
	gateway *Gateway
}

func RegisterGRPCGateway(server grpc.ServiceRegistrar, gateway *Gateway) {
	logingestv1.RegisterLogIngestServiceServer(server, &GRPCGatewayServer{gateway: gateway})
}

func (s *GRPCGatewayServer) Stream(stream grpc.BidiStreamingServer[logingestv1.LogBatch, logingestv1.IngestAck]) error {
	token := grpcBearerToken(stream.Context())
	for {
		batch, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		ack := s.gateway.Submit(stream.Context(), token, grpcBatchToModel(batch))
		if err := stream.Send(modelAckToGRPC(ack)); err != nil {
			return err
		}
	}
}

func grpcBearerToken(ctx context.Context) string {
	values, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	for _, key := range []string{"x-opshub-agent-token", "authorization"} {
		for _, value := range values.Get(key) {
			value = strings.TrimSpace(value)
			if strings.EqualFold(key, "authorization") && strings.HasPrefix(strings.ToLower(value), "bearer ") {
				value = strings.TrimSpace(value[7:])
			}
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func grpcBatchToModel(batch *logingestv1.LogBatch) LogBatch {
	result := LogBatch{
		BatchID: batch.GetBatchId(), AgentID: batch.GetAgentId(), PolicyID: batch.GetPolicyId(),
		PolicyVersion: batch.GetPolicyVersion(), SequenceStart: batch.GetSequenceStart(), SequenceEnd: batch.GetSequenceEnd(),
		SourceType: batch.GetSourceType(), AssetType: batch.GetAssetType(), AssetID: batch.GetAssetId(),
		HostID: batch.GetHostId(), ClusterID: batch.GetClusterId(), Environment: batch.GetEnvironment(),
		Service: batch.GetService(), Namespace: batch.GetNamespace(), WorkloadKind: batch.GetWorkloadKind(),
		WorkloadName: batch.GetWorkloadName(), PodName: batch.GetPodName(), PodUID: batch.GetPodUid(),
		ContainerName: batch.GetContainerName(), ContainerImage: batch.GetContainerImage(), NodeName: batch.GetNodeName(),
		FilePath: batch.GetFilePath(), Stream: batch.GetStream(), Records: make([]LogRecord, 0, len(batch.GetRecords())),
	}
	for _, record := range batch.GetRecords() {
		result.Records = append(result.Records, LogRecord{
			Sequence: record.GetSequence(), TimestampUnixNano: record.GetTimestampUnixNano(),
			ObservedTimestampUnixNano: record.GetObservedTimestampUnixNano(), Body: record.GetBody(),
			SeverityText: record.GetSeverityText(), SeverityNumber: record.GetSeverityNumber(),
			RetentionDays: int(record.GetRetentionDays()),
			Attributes:    record.GetAttributes(), ResourceAttributes: record.GetResourceAttributes(),
			TraceID: record.GetTraceId(), SpanID: record.GetSpanId(),
		})
	}
	return result
}

func modelAckToGRPC(ack IngestAck) *logingestv1.IngestAck {
	return &logingestv1.IngestAck{
		BatchId: ack.BatchID, AcceptedRecords: int32(ack.AcceptedRecords), AcceptedSequence: ack.AcceptedSequence,
		RetryAfterMs: int32(ack.RetryAfterMS), Duplicate: ack.Duplicate,
		ErrorCode: ack.ErrorCode, ErrorMessage: ack.ErrorMessage,
	}
}
