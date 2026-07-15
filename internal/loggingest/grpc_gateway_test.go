package loggingest

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	logingestv1 "github.com/ydcloud-dy/opshub/api/logingest/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestGRPCGatewayStream(t *testing.T) {
	sink := &testSink{}
	sink.failures.Store(-1)
	writer := NewWriter(WriterConfig{Workers: 1}, sink)
	defer writer.Close()
	writerServer := httptest.NewServer(WriterHTTPHandler(writer, "writer-token", 0))
	defer writerServer.Close()
	gateway := NewGateway(GatewayConfig{
		WriterURL: writerServer.URL, WriterToken: "writer-token", AgentTokens: []string{"agent-token"},
	})

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	RegisterGRPCGateway(server, gateway)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()

	connection, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer agent-token"))
	stream, err := logingestv1.NewLogIngestServiceClient(connection).Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	batch := validBatch()
	if err := stream.Send(&logingestv1.LogBatch{
		BatchId: batch.BatchID, AgentId: batch.AgentID, AssetType: batch.AssetType, AssetId: batch.AssetID,
		HostId: batch.HostID, SequenceStart: batch.SequenceStart, SequenceEnd: batch.SequenceEnd,
		Records: []*logingestv1.LogRecord{{
			Sequence: batch.Records[0].Sequence, TimestampUnixNano: batch.Records[0].TimestampUnixNano,
			Body: batch.Records[0].Body, SeverityText: batch.Records[0].SeverityText,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	ack, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if ack.GetErrorCode() != "" || ack.GetAcceptedRecords() != 1 {
		t.Fatalf("unexpected ack: %+v", ack)
	}
}
