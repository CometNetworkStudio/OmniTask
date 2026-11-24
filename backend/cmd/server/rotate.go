package main

import (
	"fmt"
	"time"

	"backend/internal/migration"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/crypto"
	"backend/pkg/log"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// rotateCredentials 把所有凭据重加密到当前主密钥版本，以便安全下线旧密钥。
// 建议停止 server 后执行，避免轮换期间同时写入。
func rotateCredentials(conf *viper.Viper, logger *log.Logger) error {
	if err := migration.Up(conf); err != nil {
		return fmt.Errorf("migrate before rotate: %w", err)
	}
	keys, err := crypto.FromConfig(conf)
	if err != nil {
		return err
	}
	db := repository.NewDB(conf, logger)
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}
	current := keys.CurrentVersion()

	var total, rotated, skipped, failed int
	var batch []model.Credential
	err = db.FindInBatches(&batch, 100, func(tx *gorm.DB, _ int) error {
		for i := range batch {
			cred := &batch[i]
			total++
			if cred.KeyVersion == current {
				skipped++
				continue
			}
			if cred.KeyVersion > current {
				failed++
				logger.Error("rotate credential uses newer key version than current",
					zap.String("credential_id", cred.ID),
					zap.Int("key_version", cred.KeyVersion),
					zap.Int("current_version", current),
				)
				continue
			}
			plain, err := keys.Decrypt(cred.Cipher, cred.Nonce, cred.KeyVersion)
			if err != nil {
				failed++
				logger.Error("rotate credential decrypt failed",
					zap.String("credential_id", cred.ID),
					zap.Int("key_version", cred.KeyVersion),
					zap.Error(err),
				)
				continue
			}
			ciphertext, nonce, version, err := keys.Encrypt(plain)
			if err != nil {
				failed++
				logger.Error("rotate credential encrypt failed",
					zap.String("credential_id", cred.ID),
					zap.Error(err),
				)
				continue
			}
			if err := tx.Model(&model.Credential{}).Where("id = ?", cred.ID).Updates(map[string]interface{}{
				"cipher":      ciphertext,
				"nonce":       nonce,
				"key_version": version,
				"updated_at":  time.Now(),
			}).Error; err != nil {
				failed++
				logger.Error("rotate credential update failed",
					zap.String("credential_id", cred.ID),
					zap.Error(err),
				)
				continue
			}
			rotated++
		}
		return nil
	}).Error
	if err != nil {
		return err
	}
	logger.Info("credential key rotation finished",
		zap.Int("current_version", current),
		zap.Int("total", total),
		zap.Int("rotated", rotated),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed),
	)
	if failed > 0 {
		return fmt.Errorf("%d credential(s) failed to rotate; old keys must be kept", failed)
	}
	return nil
}
