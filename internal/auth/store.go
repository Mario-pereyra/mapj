package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
)

type CredentialStore struct {
	path string
	key  []byte
}

type ServiceCreds struct {
	TDN        *TDNCreds        `json:"tdn,omitempty"`
	Confluence *ConfluenceCreds `json:"confluence,omitempty"`
	ProtheusProfiles map[string]*ProtheusProfile `json:"protheusProfiles,omitempty"`
	ProtheusActive   string                      `json:"protheusActive,omitempty"`
	TDSProfiles map[string]*TDSProfile `json:"tdsProfiles,omitempty"`
	TDSActive   string                 `json:"tdsActive,omitempty"`
}

type TDNCreds struct {
	BaseURL  string `json:"baseURL"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ConfluenceCreds struct {
	BaseURL  string `json:"baseURL"`
	Username string `json:"username,omitempty"`
	Token    string `json:"token"`
	// AuthType controls the HTTP auth scheme: "bearer" (PAT for Server/DC) or "basic" (email+token for Cloud).
	// Defaults to "bearer" for non-atlassian.net URLs, "basic" for atlassian.net.
	AuthType string `json:"authType,omitempty"`
}

// ProtheusProfile is a named, persisted Protheus SQL Server connection profile.
type ProtheusProfile struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// ActiveProtheusProfile returns the current active profile.
func (c *ServiceCreds) ActiveProtheusProfile() *ProtheusProfile {
	if c.ProtheusActive != "" && c.ProtheusProfiles != nil {
		if p, ok := c.ProtheusProfiles[c.ProtheusActive]; ok {
			return p
		}
	}
	return nil
}

// SetProtheusProfile adds or updates a named profile.
// If setActive is true (or no active is set), makes this the active profile.
func (c *ServiceCreds) SetProtheusProfile(p *ProtheusProfile, setActive bool) {
	if c.ProtheusProfiles == nil {
		c.ProtheusProfiles = make(map[string]*ProtheusProfile)
	}
	c.ProtheusProfiles[p.Name] = p
	if setActive || c.ProtheusActive == "" {
		c.ProtheusActive = p.Name
	}
}

// ProtheusProfileNames returns sorted profile names.
func (c *ServiceCreds) ProtheusProfileNames() []string {
	names := make([]string, 0, len(c.ProtheusProfiles))
	for name := range c.ProtheusProfiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// HasProtheusProfiles returns true if there is at least one profile.
func (c *ServiceCreds) HasProtheusProfiles() bool {
	return len(c.ProtheusProfiles) > 0
}

// TDSProfile is a named, persisted TOTVS Application Server connection profile.
type TDSProfile struct {
	Name        string `json:"name"`
	Server      string `json:"server"`
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	User        string `json:"user"`
	Password    string `json:"password"`
	Secure      bool   `json:"secure"`
}

// ActiveTDSProfile returns the current active TDS profile.
func (c *ServiceCreds) ActiveTDSProfile() *TDSProfile {
	if c.TDSActive != "" && c.TDSProfiles != nil {
		if p, ok := c.TDSProfiles[c.TDSActive]; ok {
			return p
		}
	}
	return nil
}

// SetTDSProfile adds or updates a named TDS profile.
func (c *ServiceCreds) SetTDSProfile(p *TDSProfile, setActive bool) {
	if c.TDSProfiles == nil {
		c.TDSProfiles = make(map[string]*TDSProfile)
	}
	c.TDSProfiles[p.Name] = p
	if setActive || c.TDSActive == "" {
		c.TDSActive = p.Name
	}
}

// TDSProfileNames returns sorted TDS profile names.
func (c *ServiceCreds) TDSProfileNames() []string {
	names := make([]string, 0, len(c.TDSProfiles))
	for name := range c.TDSProfiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// HasTDSProfiles returns true if there is at least one TDS profile.
func (c *ServiceCreds) HasTDSProfiles() bool {
	return len(c.TDSProfiles) > 0
}

func GetEncryptionKey() ([]byte, error) {
	if envKey := os.Getenv("MAPJ_ENCRYPTION_KEY"); envKey != "" {
		key := []byte(envKey)
		if len(key) == 32 {
			return key, nil
		}
		return nil, fmt.Errorf("MAPJ_ENCRYPTION_KEY must be exactly 32 bytes, got %d", len(key))
	}

	return deriveMachineKey()
}

func deriveMachineKey() ([]byte, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	combined := hostname + currentUser.Username
	hash := sha256.Sum256([]byte(combined))

	return hash[:], nil
}

func NewStore() (*CredentialStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}
	configDir := filepath.Join(home, ".config", "mapj")
	os.MkdirAll(configDir, 0700)

	key, err := GetEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	return &CredentialStore{
		path: filepath.Join(configDir, "credentials.enc"),
		key:  key,
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
		return creds.HasProtheusProfiles()
	case "tds":
		return creds.HasTDSProfiles()
	}
	return false
}
