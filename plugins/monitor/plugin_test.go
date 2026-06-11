package monitor

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDefaultFeishuTemplatesAreValidJSON(t *testing.T) {
	for name, raw := range map[string]string{
		"firing":  feishuFiringTemplate(),
		"recover": feishuRecoverTemplate(),
	} {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			t.Fatalf("expected %s template to be valid JSON: %v", name, err)
		}
	}
}

func TestMonitorSchedulerIntervalsSupportShortAlertRules(t *testing.T) {
	if alertRuleSchedulerInterval > 5*time.Second {
		t.Fatalf("alert rule scheduler interval should stay short enough for 10s rules, got %s", alertRuleSchedulerInterval)
	}
	if monitorMaintenanceSchedulerInterval < time.Minute {
		t.Fatalf("maintenance scheduler should not run high-frequency background checks, got %s", monitorMaintenanceSchedulerInterval)
	}
}
