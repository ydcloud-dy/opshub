package service

import (
	"strings"
	"testing"
)

func TestAppendKubernetesAccessConditionScopesClustersAndNamespaces(t *testing.T) {
	conditions := []string{"timestamp >= {start:String}"}
	params := map[string]string{}
	appendKubernetesAccessCondition(&conditions, params, map[uint64][]string{
		9: {"production", "staging"},
		3: nil,
	}, "acl")

	where := strings.Join(conditions, " AND ")
	for _, expected := range []string{
		"cluster_id = 0",
		"cluster_id = 3",
		"cluster_id = 9 AND namespace IN ({acl_0:String},{acl_1:String})",
	} {
		if !strings.Contains(where, expected) {
			t.Fatalf("Kubernetes ACL condition does not contain %q: %s", expected, where)
		}
	}
	if params["acl_0"] != "production" || params["acl_1"] != "staging" {
		t.Fatalf("namespace ACL params are incorrect: %#v", params)
	}
}

func TestApplyLogFieldSecurityHandlesNestedArrays(t *testing.T) {
	items := []LogItem{{
		Message: "plain message",
		Level:   "ERROR",
		Labels:  map[string]string{"token": "label-token", "service": "api"},
		Fields: map[string]interface{}{
			"records": []interface{}{
				map[string]interface{}{"Password": "plain-password", "user": "demo"},
				map[string]interface{}{"secret": "plain-secret"},
			},
		},
	}}

	secured := applyLogFieldSecurity(items, []string{"password", "labels.token"}, []string{"secret", "body"})
	if secured[0].Message != "******" {
		t.Fatalf("message was not masked: %#v", secured[0])
	}
	if _, exists := secured[0].Labels["token"]; exists {
		t.Fatalf("denied label was retained: %#v", secured[0].Labels)
	}
	records := secured[0].Fields["records"].([]interface{})
	first := records[0].(map[string]interface{})
	second := records[1].(map[string]interface{})
	if _, exists := first["Password"]; exists {
		t.Fatalf("denied nested field was retained: %#v", first)
	}
	if second["secret"] != "******" {
		t.Fatalf("nested array field was not masked: %#v", second)
	}
}

func TestBuildInternalWhereAppliesCollectionPolicyScope(t *testing.T) {
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-24T00:00:00Z", End: "2026-07-24T01:00:00Z",
		AllowedPolicyIDs: []uint64{12, 7, 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(built.Where, "policy_id IN (12,7,12)") {
		t.Fatalf("collection policy ACL missing from query: %s", built.Where)
	}
}

func TestBuildInternalWhereDeniesEmptyCollectionPolicyScope(t *testing.T) {
	built, err := buildInternalWhere(InternalQueryRequest{
		Start: "2026-07-24T00:00:00Z", End: "2026-07-24T01:00:00Z",
		AllowedPolicyIDs: []uint64{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(built.Where, "0 = 1") {
		t.Fatalf("empty collection policy ACL did not deny the query: %s", built.Where)
	}
}

func TestBuildInternalMetricsWhereFallsBackForCollectionPolicyScope(t *testing.T) {
	_, useMetrics, err := buildInternalMetricsWhere(InternalQueryRequest{
		Start: "2026-07-24T00:00:00Z", End: "2026-07-24T01:00:00Z",
		AllowedPolicyIDs: []uint64{7},
	})
	if err != nil {
		t.Fatal(err)
	}
	if useMetrics {
		t.Fatal("policy-scoped histogram unexpectedly used metrics without policy_id")
	}
}
