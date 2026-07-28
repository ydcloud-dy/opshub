package server

import (
	"reflect"
	"testing"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

func TestEvaluateLogAccessPoliciesDefaultsToDeny(t *testing.T) {
	if _, allowed := evaluateLogAccessPolicies(nil, "query"); allowed {
		t.Fatal("a user without an access policy was unexpectedly allowed")
	}
}

func TestLogAccessCapabilitiesFollowAllowedActions(t *testing.T) {
	policies := []logmodel.AccessPolicy{{AllowedActions: `["query","tail"]`, ScopeMode: logAccessScopeAll}}
	capabilities := logAccessCapabilitiesForPolicies(policies, false)
	if !capabilities.CanQuery || !capabilities.CanTail || capabilities.CanExport {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
	denied := logAccessCapabilitiesForPolicies(nil, false)
	if denied.CanQuery || denied.CanTail || denied.CanExport {
		t.Fatalf("missing policies should deny access: %#v", denied)
	}
	admin := logAccessCapabilitiesForPolicies(nil, true)
	if !admin.IsAdmin || !admin.CanQuery || !admin.CanTail || !admin.CanExport {
		t.Fatalf("admin capabilities were incomplete: %#v", admin)
	}
}

func TestEvaluateLogAccessPoliciesUsesActionSpecificCollectionScopes(t *testing.T) {
	policies := []logmodel.AccessPolicy{
		{AllowedActions: `["query"]`, ScopeMode: logAccessScopeCollectionPolicy, Scopes: accessPolicyScopes([]uint{9, 3})},
		{AllowedActions: `["export"]`, ScopeMode: logAccessScopeCollectionPolicy, Scopes: accessPolicyScopes([]uint{17})},
		{AllowedActions: `["tail"]`, ScopeMode: logAccessScopeAll},
	}
	queryDecision, queryAllowed := evaluateLogAccessPolicies(policies, "query")
	if !queryAllowed || !reflect.DeepEqual(queryDecision.AllowedPolicyIDs, []uint64{3, 9}) {
		t.Fatalf("unexpected query decision: %#v", queryDecision)
	}
	exportDecision, exportAllowed := evaluateLogAccessPolicies(policies, "export")
	if !exportAllowed || !reflect.DeepEqual(exportDecision.AllowedPolicyIDs, []uint64{17}) {
		t.Fatalf("unexpected export decision: %#v", exportDecision)
	}
	tailDecision, tailAllowed := evaluateLogAccessPolicies(policies, "tail")
	if !tailAllowed || tailDecision.AllowedPolicyIDs != nil {
		t.Fatalf("all scope should be unrestricted: %#v", tailDecision)
	}
}

func TestEvaluateLogAccessPoliciesUnionsScopesAndFieldRestrictions(t *testing.T) {
	policies := []logmodel.AccessPolicy{
		{AllowedActions: `["query"]`, ScopeMode: logAccessScopeCollectionPolicy, DeniedFields: `["secret"]`, Scopes: accessPolicyScopes([]uint{4})},
		{AllowedActions: `["query"]`, ScopeMode: logAccessScopeCollectionPolicy, MaskFields: `["token"]`, Scopes: accessPolicyScopes([]uint{8, 4})},
	}
	decision, allowed := evaluateLogAccessPolicies(policies, "query")
	if !allowed || !reflect.DeepEqual(decision.AllowedPolicyIDs, []uint64{4, 8}) {
		t.Fatalf("unexpected policy union: %#v", decision)
	}
	if !reflect.DeepEqual(decision.DeniedFields, []string{"secret"}) || !reflect.DeepEqual(decision.MaskFields, []string{"token"}) {
		t.Fatalf("field restrictions were not accumulated: %#v", decision)
	}
}

func TestAccessPolicyPayloadRequiresSelectedCollectionPolicy(t *testing.T) {
	_, err := accessPolicyFromPayload(accessPolicyPayload{
		Name: "limited", SubjectType: "user", SubjectID: 1,
		ScopeMode: logAccessScopeCollectionPolicy, AllowedActions: []string{"query"}, Enabled: true,
	})
	if err == nil {
		t.Fatal("selected collection policy mode accepted an empty scope")
	}
}
