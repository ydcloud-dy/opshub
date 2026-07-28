package server

import (
	"testing"
	"time"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

func TestStorageSecretDecryptsWithLegacyKey(t *testing.T) {
	t.Setenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY", "legacy-secret")
	t.Setenv("OPSHUB_LOGCENTER_DECRYPTION_KEYS", "")
	encrypted, err := encryptStorageSecret("clickhouse-password")
	if err != nil {
		t.Fatalf("encryptStorageSecret() error = %v", err)
	}

	t.Setenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY", "current-secret")
	t.Setenv("OPSHUB_LOGCENTER_DECRYPTION_KEYS", "legacy-secret")
	plainText, keyIndex, err := decryptStorageSecretWithKeyIndex(encrypted)
	if err != nil {
		t.Fatalf("decryptStorageSecretWithKeyIndex() error = %v", err)
	}
	if plainText != "clickhouse-password" || keyIndex != 1 {
		t.Fatalf("plainText = %q, keyIndex = %d", plainText, keyIndex)
	}

	rotated, err := encryptStorageSecret(plainText)
	if err != nil {
		t.Fatalf("encrypt rotated secret error = %v", err)
	}
	t.Setenv("OPSHUB_LOGCENTER_DECRYPTION_KEYS", "")
	plainText, keyIndex, err = decryptStorageSecretWithKeyIndex(rotated)
	if err != nil || plainText != "clickhouse-password" || keyIndex != 0 {
		t.Fatalf("rotated secret plainText = %q, keyIndex = %d, err = %v", plainText, keyIndex, err)
	}
}

func TestStorageSecretRejectsUnknownKey(t *testing.T) {
	t.Setenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY", "first-secret")
	t.Setenv("OPSHUB_LOGCENTER_DECRYPTION_KEYS", "")
	encrypted, err := encryptStorageSecret("clickhouse-password")
	if err != nil {
		t.Fatalf("encryptStorageSecret() error = %v", err)
	}

	t.Setenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY", "second-secret")
	if _, _, err := decryptStorageSecretWithKeyIndex(encrypted); err == nil {
		t.Fatal("decryptStorageSecretWithKeyIndex() error = nil, want unknown-key error")
	}
}

func TestStorageEncryptionKeysDeduplicateLegacyValues(t *testing.T) {
	t.Setenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY", "current-secret")
	t.Setenv("OPSHUB_LOGCENTER_DECRYPTION_KEYS", "legacy-secret,current-secret\nlegacy-secret")
	keys, err := storageEncryptionKeys()
	if err != nil {
		t.Fatalf("storageEncryptionKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2", len(keys))
	}
}

func TestStorageClusterPayloadAcceptsKafkaQueue(t *testing.T) {
	item, err := storageClusterFromPayload(storageClusterPayload{
		Name: "built-in", Endpoints: "http://clickhouse:8123", QueueMode: "kafka", Enabled: true,
	})
	if err != nil {
		t.Fatalf("storageClusterFromPayload() error = %v", err)
	}
	if item.QueueMode != "kafka" {
		t.Fatalf("QueueMode = %q, want kafka", item.QueueMode)
	}
}

func TestPrepareStorageForQueryResponseHidesConnectionDetails(t *testing.T) {
	now := time.Now()
	item := logmodel.StorageCluster{
		ID: 7, Name: "primary", StorageType: "clickhouse", Endpoints: "https://clickhouse.internal:8443",
		DatabaseName: "opshub_logs", Username: "opshub", PasswordEncrypted: "enc:v1:secret",
		SkipTLSVerify: true, Timeout: 300, QueueMode: "kafka", QueueEndpoints: "redpanda.internal:9092",
		Status: "healthy", LastError: "dial clickhouse.internal:8443", InitializedAt: &now,
		Enabled: true, IsPrimary: true, DefaultRetentionDays: 30,
	}
	prepareStorageForResponse(&item)
	prepareStorageForQueryResponse(&item)

	if item.ID != 7 || item.Name != "primary" || item.StorageType != "clickhouse" || !item.Enabled || !item.IsPrimary || item.InitializedAt == nil {
		t.Fatalf("query-safe storage identity was removed: %#v", item)
	}
	if item.Endpoints != "" || item.DatabaseName != "" || item.Username != "" || item.PasswordConfigured ||
		item.SkipTLSVerify || item.Timeout != 0 || item.QueueMode != "" || item.QueueEndpoints != "" || item.LastError != "" {
		t.Fatalf("connection details leaked from query-safe storage: %#v", item)
	}
}
