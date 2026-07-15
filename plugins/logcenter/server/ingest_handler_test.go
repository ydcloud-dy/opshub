package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
<<<<<<< HEAD
=======
	"time"
>>>>>>> feat: update log

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

<<<<<<< HEAD
=======
func TestBuildIngestReadinessReportsHealthyKafkaPipeline(t *testing.T) {
	t.Setenv("OPSHUB_LOG_AGENT_INGEST_TOKEN", "random-ingest-token")
	t.Setenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY", "random-encryption-key")
	t.Setenv("OPSHUB_LOG_AGENT_IMAGE", "registry.example.com/opshub-log-agent:v0.0.9")
	initializedAt := time.Now()
	component := ingestComponentResult{ComponentStatus: loggingest.ComponentStatus{Status: "healthy"}, Reachable: true}
	checks := buildIngestReadiness(
		component,
		component,
		ingestQueueResult{Enabled: true, Status: "healthy", Reachable: true},
		ingestStorageResult{ID: 1, Status: "healthy", Reachable: true, InitializedAt: &initializedAt},
		"https://opshub.example.com",
	)
	summary := summarizeIngestReadiness(checks)
	if summary.Passed != summary.Total || summary.Warnings != 0 || summary.Failed != 0 {
		t.Fatalf("unexpected readiness summary: %+v", summary)
	}
}

func TestBuildIngestReadinessExplainsProductionWarnings(t *testing.T) {
	t.Setenv("OPSHUB_LOG_AGENT_INGEST_TOKEN", "opshub-log-ingest-token-change-in-production")
	t.Setenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY", "opshub-logcenter-encryption-key-change-in-production")
	t.Setenv("OPSHUB_LOG_AGENT_IMAGE", "registry.example.com/opshub-log-agent:latest")
	initializedAt := time.Now()
	component := ingestComponentResult{ComponentStatus: loggingest.ComponentStatus{Status: "healthy"}, Reachable: true}
	checks := buildIngestReadiness(
		component,
		component,
		ingestQueueResult{Status: "bypassed", Reachable: true},
		ingestStorageResult{ID: 1, Status: "healthy", Reachable: true, InitializedAt: &initializedAt},
		"http://localhost:19880",
	)
	summary := summarizeIngestReadiness(checks)
	if summary.Warnings != 3 || summary.Failed != 1 {
		t.Fatalf("unexpected readiness summary: %+v", summary)
	}
	if check := readinessCheckByID(checks, "public-gateway"); check.Status != "failed" {
		t.Fatalf("public gateway check = %+v", check)
	}
}

func readinessCheckByID(checks []ingestReadinessCheck, id string) ingestReadinessCheck {
	for _, check := range checks {
		if check.ID == id {
			return check
		}
	}
	return ingestReadinessCheck{}
}

>>>>>>> feat: update log
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
