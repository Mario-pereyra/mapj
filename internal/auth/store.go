package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CredentialStore struct {
	path string
	key  []byte
}

type ServiceCreds struct {
	TDN        *TDNCreds        `json:"tdn,omitempty"`
	Confluence *ConfluenceCreds `json:"confluence,omitempty"`
	Protheus   *ProtheusCreds   `json:"protheus,omitempty"`
}

type TDNCreds struct {
	BaseURL  string `json:"baseURL"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ConfluenceCreds struct {
	BaseURL  string `json:"baseURL"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ProtheusCreds struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func NewStore() (*CredentialStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}
	configDir := filepath.Join(home, ".config", "mapj")
	os.MkdirAll(configDir, 0700)

	return &CredentialStore{
		path: filepath.Join(configDir, "credentials.enc"),
	}, nil
}

func (s *CredentialStore) SetKey(key string) error {
	s.key = []byte(key)
	return nil
}

func (s *CredentialStore) Save(creds *ServiceCreds) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	encrypted, err := s.encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, encrypted, 0600)
}

func (s *CredentialStore) Load() (*ServiceCreds, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ServiceCreds{}, nil
		}
		return nil, err
	}

	decrypted, err := s.decrypt(data)
	if err != nil {
		return nil, err
	}

	var creds ServiceCreds
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

func (s *CredentialStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *CredentialStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *CredentialStore) HasService(name string) bool {
	creds, err := s.Load()
	if err != nil {
		return false
	}
	switch name {
	case "tdn":
		return creds.TDN != nil && creds.TDN.Token != ""
	case "confluence":
		return creds.Confluence != nil && creds.Confluence.Token != ""
	case "protheus":
		return creds.Protheus != nil && creds.Protheus.Server != ""
	}
	return false
}
