package crypto

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/spf13/viper"
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

func TestKeyringVersionedDecrypt(t *testing.T) {
	key1, _ := hex.DecodeString(strings.Repeat("11", 32))
	key2, _ := hex.DecodeString(strings.Repeat("22", 32))
	ring, err := NewKeyring(map[int][]byte{1: key1, 2: key2}, 2)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"token":"secret"}`)
	ciphertext, nonce, version, err := ring.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if version != 2 || !ring.HasVersion(1) || !ring.HasVersion(2) {
		t.Fatalf("unexpected keyring state: version=%d", version)
	}
	got, err := ring.Decrypt(ciphertext, nonce, version)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("decrypt current version: %s err=%v", got, err)
	}

	oldCipher, _ := NewCipher(key1)
	oldCiphertext, oldNonce, _ := oldCipher.Encrypt(plain)
	got, err = ring.Decrypt(oldCiphertext, oldNonce, 1)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("decrypt old version: %s err=%v", got, err)
	}
	if _, err := ring.Decrypt(oldCiphertext, oldNonce, 9); err == nil {
		t.Fatal("expected error for missing key version")
	}
}

func TestKeyringRejectsMissingCurrentVersion(t *testing.T) {
	key, _ := hex.DecodeString(strings.Repeat("33", 32))
	if _, err := NewKeyring(map[int][]byte{1: key}, 2); err == nil {
		t.Fatal("expected error when current version is missing")
	}
}

func TestFromConfigKeyMapIsExclusive(t *testing.T) {
	oldKey := strings.Repeat("ab", 32)
	newKey := strings.Repeat("cd", 32)
	conf := viper.New()
	conf.Set("security.master_key", strings.Repeat("ef", 32))
	conf.Set("security.master_keys", map[string]string{"2": newKey})

	ring, err := FromConfig(conf)
	if err != nil {
		t.Fatal(err)
	}
	if ring.CurrentVersion() != 2 {
		t.Fatalf("current version: %d", ring.CurrentVersion())
	}
	if ring.HasVersion(1) {
		t.Fatal("master_key should be ignored when master_keys is non-empty")
	}

	conf2 := viper.New()
	conf2.Set("security.master_keys", map[string]string{"1": oldKey, "2": newKey})
	ring2, err := FromConfig(conf2)
	if err != nil {
		t.Fatal(err)
	}
	if !ring2.HasVersion(1) || ring2.CurrentVersion() != 2 {
		t.Fatalf("unexpected keyring: current=%d", ring2.CurrentVersion())
	}
}
