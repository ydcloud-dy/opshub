package service

import "testing"

func TestSecretCipherRoundTrip(t *testing.T) {
	cipher, err := NewSecretCipher("a-long-application-inventory-master-key")
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	plaintext := []byte(`{"password":"p@ss","token":"token-value"}`)
	sealed, err := cipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if sealed == string(plaintext) || sealed == "" {
		t.Fatalf("ciphertext should not expose plaintext")
	}
	opened, err := cipher.Decrypt(sealed)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(opened) != string(plaintext) {
		t.Fatalf("round trip mismatch: got %q", opened)
	}
}

func TestSecretCipherRejectsWrongKey(t *testing.T) {
	first, err := NewSecretCipher("first-master-key")
	if err != nil {
		t.Fatalf("create first cipher: %v", err)
	}
	second, err := NewSecretCipher("second-master-key")
	if err != nil {
		t.Fatalf("create second cipher: %v", err)
	}
	sealed, err := first.Encrypt([]byte("private-value"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := second.Decrypt(sealed); err == nil {
		t.Fatal("expected wrong key decryption to fail")
	}
}

func TestSecretCipherRequiresConfiguredKey(t *testing.T) {
	if _, err := NewSecretCipher(""); err == nil {
		t.Fatal("expected empty key to be rejected")
	}
}

func TestCredentialSecretEmpty(t *testing.T) {
	if !(CredentialSecret{}).Empty() {
		t.Fatal("empty credential should be detected")
	}
	if (CredentialSecret{Password: "value"}).Empty() {
		t.Fatal("credential with password should not be empty")
	}
}
