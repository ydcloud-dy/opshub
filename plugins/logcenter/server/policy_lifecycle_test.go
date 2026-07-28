package server

import (
	"strings"
	"testing"
	"time"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

func TestCollectionPolicyDeleteAction(t *testing.T) {
	tests := []struct {
		name       string
		policy     logmodel.CollectionPolicy
		expected   string
		errorMatch string
	}{
		{name: "unpublished draft is deleted", policy: logmodel.CollectionPolicy{Status: collectionPolicyStatusDraft}, expected: "deleted"},
		{name: "published draft is not deleted", policy: logmodel.CollectionPolicy{Status: collectionPolicyStatusDraft, Version: 1}, errorMatch: "当前策略状态不允许删除"},
		{name: "disabled policy is archived", policy: logmodel.CollectionPolicy{Status: collectionPolicyStatusDisabled, Version: 3}, expected: "archived"},
		{name: "published policy must be disabled first", policy: logmodel.CollectionPolicy{Status: collectionPolicyStatusPublished, Version: 2}, errorMatch: "先停用"},
		{name: "archived policy is idempotently protected", policy: logmodel.CollectionPolicy{Status: collectionPolicyStatusArchived, Version: 2}, errorMatch: "已经归档"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			action, err := collectionPolicyDeleteAction(test.policy)
			if test.errorMatch != "" {
				if err == nil || !strings.Contains(err.Error(), test.errorMatch) {
					t.Fatalf("expected error containing %q, got action=%q err=%v", test.errorMatch, action, err)
				}
				return
			}
			if err != nil || action != test.expected {
				t.Fatalf("unexpected delete decision: action=%q err=%v expected=%q", action, err, test.expected)
			}
		})
	}
}

func TestCollectorInstanceRuntimeAndLifecycleStatus(t *testing.T) {
	now := time.Now()
	recent := now.Add(-30 * time.Second)
	old := now.Add(-2 * time.Minute)

	if status := collectorRuntimeStatus(logmodel.CollectorInstance{Status: "online", LastHeartbeatAt: &old}, now); status != "offline" {
		t.Fatalf("stale heartbeat should be offline, got %q", status)
	}
	if status := collectorRuntimeStatus(logmodel.CollectorInstance{Status: "offline", LastHeartbeatAt: &recent}, now); status != "online" {
		t.Fatalf("recent heartbeat should be online, got %q", status)
	}

	cases := []struct {
		name              string
		instance          logmodel.CollectorInstance
		credentialStatus  string
		activePolicyCount int64
		expected          string
	}{
		{name: "host collectors are active", instance: logmodel.CollectorInstance{Mode: "host"}, expected: "active"},
		{name: "revoked kubernetes collector is retired", instance: logmodel.CollectorInstance{Mode: "kubernetes-node"}, credentialStatus: "revoked", activePolicyCount: 1, expected: "retired"},
		{name: "installed kubernetes collector without policies is idle", instance: logmodel.CollectorInstance{Mode: "kubernetes-node"}, credentialStatus: "active", expected: "idle"},
		{name: "kubernetes collector with published policies is active", instance: logmodel.CollectorInstance{Mode: "kubernetes-node"}, credentialStatus: "active", activePolicyCount: 2, expected: "active"},
		{name: "old kubernetes collector without credential is retired", instance: logmodel.CollectorInstance{Mode: "kubernetes-node"}, expected: "retired"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			status := collectorLifecycleStatus(test.instance, "offline", test.credentialStatus, test.activePolicyCount)
			if status != test.expected {
				t.Fatalf("unexpected lifecycle status: got %q want %q", status, test.expected)
			}
		})
	}
}
