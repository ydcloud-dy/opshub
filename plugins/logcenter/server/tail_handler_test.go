package server

import (
	"testing"
	"time"

	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
)

func TestApplyTailAccessDecisionRefreshesRestrictions(t *testing.T) {
	req := logsvc.InternalQueryRequest{
		AllowedPolicyIDs: []uint64{1}, AllowedHostIDs: []uint64{2},
		AllowedKubernetesScopes: map[uint64][]string{3: {"old"}},
		DeniedFields:            []string{"oldDenied"}, MaskFields: []string{"oldMasked"},
	}
	applyTailAccessDecision(&req, logAccessDecision{
		AllowedPolicyIDs: []uint64{4}, AllowedHostIDs: []uint64{5},
		AllowedKubernetesScopes: map[uint64][]string{6: {"production"}},
		DeniedFields:            []string{"authorization"}, MaskFields: []string{"phone"},
	})
	if len(req.AllowedPolicyIDs) != 1 || req.AllowedPolicyIDs[0] != 4 || len(req.AllowedHostIDs) != 1 || req.AllowedHostIDs[0] != 5 {
		t.Fatalf("tail restrictions were not refreshed: %#v", req)
	}
	if req.AllowedKubernetesScopes[6][0] != "production" || req.DeniedFields[0] != "authorization" || req.MaskFields[0] != "phone" {
		t.Fatalf("tail field or Kubernetes restrictions were not refreshed: %#v", req)
	}
	applyTailAccessDecision(&req, logAccessDecision{IsAdmin: true})
	if req.AllowedPolicyIDs != nil || req.AllowedHostIDs != nil || req.AllowedKubernetesScopes != nil || req.DeniedFields != nil || req.MaskFields != nil {
		t.Fatalf("administrator tail restrictions were not cleared: %#v", req)
	}
}

func TestTailStreamDeadlineUsesTokenExpiry(t *testing.T) {
	now := time.Date(2026, 7, 27, 6, 0, 0, 0, time.UTC)
	duration, reason := tailStreamDeadline(24*time.Hour, now.Add(90*time.Second), now)
	if duration != 90*time.Second || reason != "token_expired" {
		t.Fatalf("deadline = %v, reason = %q", duration, reason)
	}
	duration, reason = tailStreamDeadline(time.Hour, now.Add(2*time.Hour), now)
	if duration != time.Hour || reason != "max_duration" {
		t.Fatalf("deadline = %v, reason = %q", duration, reason)
	}
}

func TestTailReauthorizationIntervalIsBounded(t *testing.T) {
	t.Setenv("OPSHUB_LOG_TAIL_REAUTH_SECONDS", "1")
	if interval := tailReauthorizationInterval(); interval != 5*time.Second {
		t.Fatalf("minimum interval = %v", interval)
	}
	t.Setenv("OPSHUB_LOG_TAIL_REAUTH_SECONDS", "999")
	if interval := tailReauthorizationInterval(); interval != 300*time.Second {
		t.Fatalf("maximum interval = %v", interval)
	}
}
