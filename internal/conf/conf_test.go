package conf

import (
	"strings"
	"testing"
	"time"
)

func TestConfigureLocalTimezoneFromEnv(t *testing.T) {
	oldLocal := time.Local
	defer func() { time.Local = oldLocal }()
	t.Setenv("OPSHUB_SERVER_TIMEZONE", "Asia/Shanghai")

	if err := configureLocalTimezone(""); err != nil {
		t.Fatalf("configureLocalTimezone() error = %v", err)
	}
	_, offset := time.Date(2026, 6, 17, 12, 0, 0, 0, time.Local).Zone()
	if offset != 8*60*60 {
		t.Fatalf("expected Asia/Shanghai offset, got %d", offset)
	}
}

func TestDatabaseDSNIncludesSessionTimezone(t *testing.T) {
	oldLocal := time.Local
	defer func() { time.Local = oldLocal }()
	time.Local = time.FixedZone("Asia/Shanghai", 8*60*60)

	cfg := DatabaseConfig{
		Host:     "mysql",
		Port:     3306,
		Database: "opshub",
		Username: "root",
		Password: "secret",
	}
	dsn := cfg.GetDSN()
	if !strings.Contains(dsn, "loc=Local") {
		t.Fatalf("expected loc=Local in DSN, got %s", dsn)
	}
	if !strings.Contains(dsn, "time_zone=%27%2B08%3A00%27") {
		t.Fatalf("expected MySQL session time_zone +08:00 in DSN, got %s", dsn)
	}
}
