package service

import (
	"context"
	"encoding/json"
	"errors"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/crypto"
	"backend/pkg/log"
	"backend/pkg/sid"
	"gorm.io/gorm"
)

type CredentialService struct {
	repo   *repository.CredentialRepository
	cipher *crypto.Cipher
	sid    *sid.Sid
	logger *log.Logger
}

func NewCredentialService(repo *repository.CredentialRepository, cipher *crypto.Cipher, sid *sid.Sid, logger *log.Logger) *CredentialService {
	return &CredentialService{repo: repo, cipher: cipher, sid: sid, logger: logger}
}

func (s *CredentialService) Upsert(ctx context.Context, userID string, req *v1.CredentialUpsertRequest) (*v1.CredentialResponse, error) {
	plain, err := json.Marshal(req.Data)
	if err != nil {
		return nil, err
	}
	ciphertext, nonce, err := s.cipher.Encrypt(plain)
	if err != nil {
		return nil, v1.ErrCredentialEncrypt
	}
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	cred := &model.Credential{
		ID:       id,
		UserID:   userID,
		Platform: req.Platform,
		Cipher:   ciphertext,
		Nonce:    nonce,
	}
	if err := s.repo.Upsert(ctx, cred); err != nil {
		return nil, err
	}
	saved, err := s.repo.Get(ctx, userID, req.Platform)
	if err != nil {
		return nil, err
	}
	return toCredentialResponse(saved), nil
}

func (s *CredentialService) List(ctx context.Context, userID string) ([]v1.CredentialResponse, error) {
	creds, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]v1.CredentialResponse, 0, len(creds))
	for i := range creds {
		out = append(out, *toCredentialResponse(&creds[i]))
	}
	return out, nil
}

// Decrypted 供 M3 下发 Agent 前解密使用；明文不得写入日志。
func (s *CredentialService) Decrypted(ctx context.Context, userID, platform string) (map[string]string, error) {
	cred, err := s.repo.Get(ctx, userID, platform)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrCredentialNotFound
		}
		return nil, err
	}
	plain, err := s.cipher.Decrypt(cred.Cipher, cred.Nonce)
	if err != nil {
		return nil, v1.ErrCredentialDecrypt
	}
	data := map[string]string{}
	if err := json.Unmarshal(plain, &data); err != nil {
		return nil, v1.ErrCredentialDecrypt
	}
	return data, nil
}

func toCredentialResponse(cred *model.Credential) *v1.CredentialResponse {
	return &v1.CredentialResponse{
		ID:        cred.ID,
		Platform:  cred.Platform,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	}
}
