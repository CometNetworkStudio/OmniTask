package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// ParseMasterKey 接受 32 字节密钥：hex(64 字符)、base64 或原始 32 字符。
func ParseMasterKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("master key is empty")
	}
	if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	if len(s) == 32 {
		return []byte(s), nil
	}
	return nil, fmt.Errorf("master key must be 32 bytes as hex/base64/raw, got %d chars", len(s))
}

// Cipher 使用 AES-256-GCM 加解密凭据；nonce 随密文一起存储。
type Cipher struct {
	aead cipher.AEAD
}

func NewCipher(key []byte) (*Cipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// FromEnv 读取 OMNITASK_MASTER_KEY。
func FromEnv() (*Cipher, error) {
	key, err := ParseMasterKey(os.Getenv("OMNITASK_MASTER_KEY"))
	if err != nil {
		return nil, fmt.Errorf("OMNITASK_MASTER_KEY: %w", err)
	}
	return NewCipher(key)
}

// FromConfig 优先读配置 security.master_key，其次环境变量 OMNITASK_MASTER_KEY。
func FromConfig(conf *viper.Viper) (*Cipher, error) {
	keyStr := conf.GetString("security.master_key")
	if keyStr == "" {
		keyStr = os.Getenv("OMNITASK_MASTER_KEY")
	}
	key, err := ParseMasterKey(keyStr)
	if err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	return NewCipher(key)
}

func (c *Cipher) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return c.aead.Seal(nil, nonce, plaintext, nil), nonce, nil
}

func (c *Cipher) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	if len(nonce) != c.aead.NonceSize() {
		return nil, errors.New("invalid nonce size")
	}
	return c.aead.Open(nil, nonce, ciphertext, nil)
}
