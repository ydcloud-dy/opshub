package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/loggingest"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

func TestResolveIngestTestTokenUsesExplicitToken(t *testing.T) {
	t.Setenv("OPSHUB_LOG_INGEST_TEST_TOKEN", "test-token")
	t.Setenv("OPSHUB_LOG_INGEST_TOKEN", "shared-token")
	if token := resolveIngestTestToken("http://127.0.0.1:9880"); token != "test-token" {
		t.Fatalf("token = %q", token)
	}
}

func TestResolveIngestTestTokenAllowsLoopbackDevelopment(t *testing.T) {
	t.Setenv("OPSHUB_LOG_INGEST_TEST_TOKEN", "")
	t.Setenv("OPSHUB_LOG_INGEST_TOKEN", "")
	if token := resolveIngestTestToken("http://localhost:9880"); token != "opshub-local-ingest-token" {
		t.Fatalf("token = %q", token)
	}
}

func TestResolveIngestTestTokenDoesNotDefaultForRemoteGateway(t *testing.T) {
	t.Setenv("OPSHUB_LOG_INGEST_TEST_TOKEN", "")
	t.Setenv("OPSHUB_LOG_INGEST_TOKEN", "")
	if token := resolveIngestTestToken("https://logs.example.com"); token != "" {
		t.Fatalf("token = %q", token)
	}
}

func TestAgentGatewayURLUsesReachableFallbackForLoopbackHost(t *testing.T) {
	if gatewayURL := agentGatewayURLForHost("http", "localhost:9876", "http://192.168.31.190:9880"); gatewayURL != "http://192.168.31.190:9880" {
		t.Fatalf("gateway URL = %q", gatewayURL)
	}
}

func TestAgentGatewayURLKeepsForwardedPublicHost(t *testing.T) {
	if gatewayURL := agentGatewayURLForHost("https", "opshub.example.com", "http://192.168.31.190:9880"); gatewayURL != "https://opshub.example.com" {
		t.Fatalf("gateway URL = %q", gatewayURL)
	}
}

func TestAgentGatewayURLRewritesDirectBackendPort(t *testing.T) {
	t.Setenv("OPSHUB_SERVER_HTTP_PORT", "9876")
	if gatewayURL := agentGatewayURLForHost("http", "192.168.31.190:9876", "http://192.168.31.190:9880"); gatewayURL != "http://192.168.31.190:9880" {
		t.Fatalf("gateway URL = %q", gatewayURL)
	}
}

func TestAgentGatewayURLRewritesLocalFrontendPort(t *testing.T) {
	if gatewayURL := agentGatewayURLForHost("http", "192.168.31.190:5173", "http://192.168.31.190:9880"); gatewayURL != "http://192.168.31.190:9880" {
		t.Fatalf("gateway URL = %q", gatewayURL)
	}
}

func TestIngestSendsBatchAndReturnsGatewayAck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/logs/batches" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if token := request.Header.Get("Authorization"); token != "Bearer test-token" {
			t.Fatalf("authorization = %q", token)
		}
		var batch loggingest.LogBatch
		if err := json.NewDecoder(request.Body).Decode(&batch); err != nil {
			t.Fatalf("decode batch: %v", err)
		}
		if len(batch.Records) != 1 || batch.Records[0].Body != "integration message" {
			t.Fatalf("unexpected records: %+v", batch.Records)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(loggingest.IngestAck{
			BatchID: batch.BatchID, AcceptedRecords: 1, AcceptedSequence: batch.SequenceEnd,
		})
	}))
	defer gateway.Close()
	t.Setenv("OPSHUB_LOG_GATEWAY_STATUS_URL", gateway.URL)
	t.Setenv("OPSHUB_LOG_INGEST_TEST_TOKEN", "test-token")

	router := gin.New()
	handler := &Handler{}
	router.POST("/test", handler.TestIngest)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"message":"integration message","level":"WARN"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != 0 {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}
