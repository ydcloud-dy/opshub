package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

func TestTemplateAccessAllowed(t *testing.T) {
	privateTemplate := logmodel.QueryTemplate{OwnerID: 10}
	publicTemplate := logmodel.QueryTemplate{OwnerID: 10, IsPublic: true}

	if !templateAccessAllowed(privateTemplate, 10, false, true) {
		t.Fatal("owner cannot update private template")
	}
	if templateAccessAllowed(privateTemplate, 20, false, false) {
		t.Fatal("other user can read private template")
	}
	if templateAccessAllowed(publicTemplate, 20, false, true) {
		t.Fatal("other user can update public template")
	}
	if !templateAccessAllowed(publicTemplate, 20, false, false) {
		t.Fatal("other user cannot read public template")
	}
	if !templateAccessAllowed(privateTemplate, 20, true, true) {
		t.Fatal("administrator cannot update template")
	}
}

func TestGlobalLogManagementEndpointsRejectNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{}
	tests := []struct {
		name    string
		method  string
		path    string
		handler gin.HandlerFunc
		message string
	}{
		{name: "overview", method: http.MethodGet, path: "/overview", handler: handler.GetOverview, message: "只有管理员可以查看日志总览"},
		{name: "ingest status", method: http.MethodGet, path: "/ingest/status", handler: handler.GetIngestStatus, message: "只有管理员可以查看日志采集链路"},
		{name: "ingest test", method: http.MethodPost, path: "/ingest/test", handler: handler.TestIngest, message: "只有管理员可以写入采集链路测试日志"},
		{name: "retention policies", method: http.MethodGet, path: "/retention-policies", handler: handler.ListRetentionPolicies, message: "只有管理员可以管理日志保留策略"},
		{name: "library", method: http.MethodGet, path: "/library", handler: handler.ListLibrary, message: "只有管理员可以管理日志库"},
		{name: "alert context", method: http.MethodGet, path: "/alerts/1/context", handler: handler.GetAlertContext, message: "只有管理员可以管理日志告警上下文"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Handle(test.method, test.path, test.handler)
			recorder := httptest.NewRecorder()
			body := strings.NewReader("{}")
			request := httptest.NewRequest(test.method, test.path, body)
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			var result response.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if result.Code != http.StatusForbidden || result.Message != test.message {
				t.Fatalf("response = %#v", result)
			}
		})
	}
}
