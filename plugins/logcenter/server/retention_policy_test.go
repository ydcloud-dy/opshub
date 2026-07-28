package server

import (
	"testing"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

func TestApplyRetentionPolicyValuesOverridesCollectionSnapshot(t *testing.T) {
	payload := policyPayload{RetentionDays: 14}
	policy := logmodel.RetentionPolicy{
		Name:        "production",
		DefaultDays: 30,
		LevelDays:   `{"TRACE":7,"ERROR":90,"FATAL":180}`,
	}
	if err := applyRetentionPolicyValues(&payload, policy); err != nil {
		t.Fatalf("apply retention policy: %v", err)
	}
	if payload.RetentionDays != 30 || payload.Retention.DefaultDays != 30 {
		t.Fatalf("unexpected default retention snapshot: %+v", payload.Retention)
	}
	if payload.Retention.LevelDays["TRACE"] != 7 || payload.Retention.LevelDays["ERROR"] != 90 || payload.Retention.LevelDays["FATAL"] != 180 {
		t.Fatalf("unexpected level retention snapshot: %+v", payload.Retention.LevelDays)
	}
}

func TestApplyRetentionPolicyValuesRejectsInvalidProfile(t *testing.T) {
	payload := policyPayload{}
	policy := logmodel.RetentionPolicy{Name: "invalid", DefaultDays: 0, LevelDays: `{}`}
	if err := applyRetentionPolicyValues(&payload, policy); err == nil {
		t.Fatal("expected invalid default retention to fail")
	}
	policy.DefaultDays = 30
	policy.LevelDays = `{invalid}`
	if err := applyRetentionPolicyValues(&payload, policy); err == nil {
		t.Fatal("expected invalid level retention JSON to fail")
	}
}

func TestRetentionPolicyViewIncludesBindingState(t *testing.T) {
	policy := logmodel.RetentionPolicy{ID: 8, Name: "production", DefaultDays: 30, LevelDays: `{"ERROR":90}`, Enabled: true}
	view := retentionPolicyToView(policy, 3, 2)
	if view.BoundPolicyCount != 3 || view.UpdatedPolicyCount != 2 {
		t.Fatalf("unexpected binding state: %+v", view)
	}
	if view.Payload.LevelDays["ERROR"] != 90 {
		t.Fatalf("unexpected level days: %+v", view.Payload.LevelDays)
	}
}
