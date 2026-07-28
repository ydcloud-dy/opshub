//go:build linux

package main

import (
	"testing"

	"github.com/ydcloud-dy/opshub/internal/logagent"
)

func TestManagedLogConfigUnchanged(t *testing.T) {
	config := logagent.Config{Enabled: true, GatewayURL: "https://logs.example.com", GatewayToken: "token"}
	config.Normalize()
	if !managedLogConfigUnchanged(config, config, 4, 4, 2, 2) {
		t.Fatal("equivalent config should not restart the collector")
	}
	changed := config
	changed.BatchSize++
	if managedLogConfigUnchanged(config, changed, 4, 4, 2, 2) {
		t.Fatal("config content change must restart the collector")
	}
	if managedLogConfigUnchanged(config, config, 4, 5, 2, 2) {
		t.Fatal("config version change must restart the collector")
	}
	if managedLogConfigUnchanged(config, config, 4, 4, 2, 3) {
		t.Fatal("reload generation change must restart the collector")
	}
}
