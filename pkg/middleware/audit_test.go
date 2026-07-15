package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetRequestParamsRedactsNestedArraysCaseInsensitively(t *testing.T) {
	body := `[{"name":"collector","Password":"plain-password","targets":[{"access_token":"plain-token","config":{"ClientSecret":"plain-secret","region":"cn"}}]}]`
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("POST", "/api/v1/plugins/logcenter/policies", strings.NewReader(body))

	filtered := getRequestParams(context, []byte(body))
	for _, secret := range []string{"plain-password", "plain-token", "plain-secret"} {
		if strings.Contains(filtered, secret) {
			t.Fatalf("sensitive value %q leaked into audit params: %s", secret, filtered)
		}
	}
	var decoded []map[string]interface{}
	if err := json.Unmarshal([]byte(filtered), &decoded); err != nil {
		t.Fatalf("filtered params are invalid JSON: %v", err)
	}
	if decoded[0]["Password"] != "******" {
		t.Fatalf("Password was not redacted: %#v", decoded[0])
	}
}

func TestGetRequestParamsRedactsSensitiveQueryValues(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/api/v1/logs?keyword=error&ApiToken=plain-token", nil)

	filtered := getRequestParams(context, nil)
	if strings.Contains(filtered, "plain-token") || !strings.Contains(filtered, "ApiToken=%2A%2A%2A%2A%2A%2A") {
		t.Fatalf("query credentials were not redacted: %s", filtered)
	}
	if !strings.Contains(filtered, "keyword=error") {
		t.Fatalf("non-sensitive query value was lost: %s", filtered)
	}
}
