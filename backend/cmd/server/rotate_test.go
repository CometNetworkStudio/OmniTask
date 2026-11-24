package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/crypto"
	"backend/pkg/log"

	"github.com/spf13/viper"
)

func TestRotateCredentials(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.ToSlash(filepath.Join(dir, "rotate.db"))
	oldKey := strings.Repeat("11", 32)
	newKey := strings.Repeat("22", 32)

	conf := viper.New()
	conf.Set("mode", "lite")
	conf.Set("data.db.user.driver", "sqlite")
	conf.Set("data.db.user.dsn", dsn)
	conf.Set("log.log_level", "error")
	conf.Set("log.mode", "console")
	conf.Set("log.encoding", "console")
	conf.Set("security.master_keys", map[string]string{"1": oldKey, "2": newKey})
	logger := log.NewLog(conf)

	db := repository.NewDB(conf, logger)
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	oldKeyBytes, err := crypto.ParseMasterKey(oldKey)
	if err != nil {
		t.Fatal(err)
	}
	oldRing, err := crypto.NewKeyring(map[int][]byte{1: oldKeyBytes}, 1)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"token":"plain-token"}`)
	ciphertext, nonce, version, err := oldRing.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("unexpected version: %d", version)
	}
	cred := &model.Credential{
		ID: "c1", UserID: "u1", Platform: "demo",
		KeyVersion: 1, Cipher: ciphertext, Nonce: nonce,
	}
	if err := db.Create(cred).Error; err != nil {
		t.Fatal(err)
	}

	if err := rotateCredentials(conf, logger); err != nil {
		t.Fatal(err)
	}

	var rotated model.Credential
	if err := db.Where("id = ?", "c1").First(&rotated).Error; err != nil {
		t.Fatal(err)
	}
	if rotated.KeyVersion != 2 {
		t.Fatalf("key version not rotated: %d", rotated.KeyVersion)
	}
	ring, err := crypto.FromConfig(conf)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ring.Decrypt(rotated.Cipher, rotated.Nonce, rotated.KeyVersion)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]string
	if err := json.Unmarshal(got, &data); err != nil || data["token"] != "plain-token" {
		t.Fatalf("unexpected decrypted data: %s err=%v", got, err)
	}

	// 旧密钥删除后仍能用新密钥解出轮换后的数据。
	conf2 := viper.New()
	conf2.Set("security.master_keys", map[string]string{"2": newKey})
	ring2, err := crypto.FromConfig(conf2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ring2.Decrypt(rotated.Cipher, rotated.Nonce, rotated.KeyVersion); err != nil {
		t.Fatalf("decrypt with new key only: %v", err)
	}
}
