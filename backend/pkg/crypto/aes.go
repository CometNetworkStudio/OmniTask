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
	"sort"
	"strconv"
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

// Keyring 保存当前主密钥与历史主密钥，支持按凭据记录的版本解密。
// 新写入始终使用当前版本（版本号最大，或被 security.master_key_version 指定）。
type Keyring struct {
	current int
	keys    map[int]*Cipher
}

func NewKeyring(keys map[int][]byte, current int) (*Keyring, error) {
	if len(keys) == 0 {
		return nil, errors.New("no master keys configured")
	}
	out := make(map[int]*Cipher, len(keys))
	versions := make([]int, 0, len(keys))
	for version, key := range keys {
		if version <= 0 {
			return nil, fmt.Errorf("invalid master key version: %d", version)
		}
		c, err := NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("master key version %d: %w", version, err)
		}
		out[version] = c
		versions = append(versions, version)
	}
	sort.Ints(versions)
	if current <= 0 {
		current = versions[len(versions)-1]
	}
	if _, ok := out[current]; !ok {
		return nil, fmt.Errorf("current master key version %d is not configured", current)
	}
	return &Keyring{current: current, keys: out}, nil
}

// FromEnv 读取 OMNITASK_MASTER_KEY（版本 1）。
func FromEnv() (*Keyring, error) {
	key, err := ParseMasterKey(os.Getenv("OMNITASK_MASTER_KEY"))
	if err != nil {
		return nil, fmt.Errorf("OMNITASK_MASTER_KEY: %w", err)
	}
	return NewKeyring(map[int][]byte{1: key}, 1)
}

// FromConfig 读取主密钥：
//   - security.master_keys 非空时只使用该映射（版本号 -> 密钥），用于轮换；
//     轮换时务必把旧 master_key 以原版本号加入映射，否则旧数据无法解密；
//   - 映射为空时读取 security.master_key，其次环境变量 OMNITASK_MASTER_KEY，版本默认 1。
func FromConfig(conf *viper.Viper) (*Keyring, error) {
	raw := conf.GetStringMapString("security.master_keys")
	keys := make(map[int][]byte, len(raw))
	for versionText, keyText := range raw {
		version, err := strconv.Atoi(strings.TrimSpace(versionText))
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("security.master_keys 的版本号必须是正整数，当前为 %q", versionText)
		}
		key, err := ParseMasterKey(keyText)
		if err != nil {
			return nil, fmt.Errorf("security.master_keys[%d]: %w", version, err)
		}
		keys[version] = key
	}

	if len(keys) == 0 {
		keyText := conf.GetString("security.master_key")
		if keyText == "" {
			keyText = os.Getenv("OMNITASK_MASTER_KEY")
		}
		if keyText == "" {
			return nil, errors.New("master key: security.master_key 或 security.master_keys 至少配置一项")
		}
		key, err := ParseMasterKey(keyText)
		if err != nil {
			return nil, fmt.Errorf("master key: %w", err)
		}
		version := conf.GetInt("security.master_key_version")
		if version <= 0 {
			version = 1
		}
		keys[version] = key
	}
	return NewKeyring(keys, conf.GetInt("security.master_key_version"))
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

func (k *Keyring) CurrentVersion() int {
	return k.current
}

func (k *Keyring) HasVersion(version int) bool {
	_, ok := k.keys[version]
	return ok
}

func (k *Keyring) Encrypt(plaintext []byte) (ciphertext, nonce []byte, version int, err error) {
	ciphertext, nonce, err = k.keys[k.current].Encrypt(plaintext)
	return ciphertext, nonce, k.current, err
}

// Decrypt 使用密文所属的密钥版本解密；版本 0 视为 1，兼容升级前的数据。
func (k *Keyring) Decrypt(ciphertext, nonce []byte, version int) ([]byte, error) {
	if version <= 0 {
		version = 1
	}
	c, ok := k.keys[version]
	if !ok {
		return nil, fmt.Errorf("master key version %d is not configured", version)
	}
	return c.Decrypt(ciphertext, nonce)
}
