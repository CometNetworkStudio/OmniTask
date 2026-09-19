package crypto

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestParseMasterKey(t *testing.T) {
	hexKey := strings.Repeat("ab", 32)
	if _, err := ParseMasterKey(hexKey); err != nil {
		t.Fatalf("hex key: %v", err)
	}
	if _, err := ParseMasterKey("short"); err == nil {
		t.Fatal("expected error for short key")
	}
	raw := strings.Repeat("x", 32)
	if k, err := ParseMasterKey(raw); err != nil || len(k) != 32 {
		t.Fatalf("raw key: %v len=%d", err, len(k))
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, _ := hex.DecodeString(strings.Repeat("cd", 32))
	c, err := NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"token":"secret-value"}`)
	ciphertext, nonce, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("secret-value")) {
		t.Fatal("ciphertext leaks plaintext")
	}
	got, err := c.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("roundtrip mismatch: %s", got)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	key1, _ := hex.DecodeString(strings.Repeat("01", 32))
	key2, _ := hex.DecodeString(strings.Repeat("02", 32))
	c1, _ := NewCipher(key1)
	c2, _ := NewCipher(key2)
	ciphertext, nonce, _ := c1.Encrypt([]byte("hello"))
	if _, err := c2.Decrypt(ciphertext, nonce); err == nil {
		t.Fatal("expected decrypt failure with wrong key")
	}
}

func TestDecryptRejectsBadNonce(t *testing.T) {
	key, _ := hex.DecodeString(strings.Repeat("ee", 32))
	c, _ := NewCipher(key)
	ciphertext, _, _ := c.Encrypt([]byte("hello"))
	if _, err := c.Decrypt(ciphertext, []byte("short")); err == nil {
		t.Fatal("expected error for invalid nonce size")
	}
}
