package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialStore_EncryptDecrypt(t *testing.T) {
	tmpDir := t.TempDir()
	key, err := GetEncryptionKey()
	require.NoError(t, err)

	store := &CredentialStore{
		path: filepath.Join(tmpDir, "credentials.enc"),
		key:  key,
	}

	creds := &ServiceCreds{
		TDN: &TDNCreds{
			BaseURL: "https://tdninterno.totvs.com",
			Token:   "test-token-123",
		},
	}

	err = store.Save(creds)
	require.NoError(t, err)

	_, err = os.Stat(store.path)
	assert.NoError(t, err)

	loaded, err := store.Load()
	require.NoError(t, err)

	assert.Equal(t, creds.TDN.BaseURL, loaded.TDN.BaseURL)
	assert.Equal(t, creds.TDN.Token, loaded.TDN.Token)
}

func TestCredentialStore_HasService(t *testing.T) {
	tmpDir := t.TempDir()
	key, err := GetEncryptionKey()
	require.NoError(t, err)

	store := &CredentialStore{
		path: filepath.Join(tmpDir, "credentials.enc"),
		key:  key,
	}

	assert.False(t, store.HasService("tdn"))
	assert.False(t, store.HasService("confluence"))
	assert.False(t, store.HasService("protheus"))

	creds := &ServiceCreds{
		TDN: &TDNCreds{BaseURL: "https://tdn.totvs.com", Token: "token"},
	}
	err = store.Save(creds)
	require.NoError(t, err)

	assert.True(t, store.HasService("tdn"))
	assert.False(t, store.HasService("confluence"))
}

func TestCredentialStore_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	key, err := GetEncryptionKey()
	require.NoError(t, err)

	store := &CredentialStore{
		path: filepath.Join(tmpDir, "nonexistent.enc"),
		key:  key,
	}

	creds, err := store.Load()
	require.NoError(t, err)
	assert.NotNil(t, creds)
}

func TestTDNCreds(t *testing.T) {
	creds := &TDNCreds{
		BaseURL: "https://tdninterno.totvs.com",
		Token:   "my-token",
	}

	assert.Equal(t, "https://tdninterno.totvs.com", creds.BaseURL)
	assert.Equal(t, "my-token", creds.Token)
}

func TestConfluenceCreds(t *testing.T) {
	creds := &ConfluenceCreds{
		BaseURL: "https://company.atlassian.net",
		Token:   "api-token",
	}

	assert.Equal(t, "https://company.atlassian.net", creds.BaseURL)
	assert.Equal(t, "api-token", creds.Token)
}

func TestProtheusProfile(t *testing.T) {
	creds := &ProtheusProfile{
		Name:     "default",
		Server:   "192.168.1.100",
		Port:     1433,
		Database: "PROTHEUS",
		User:     "admin",
		Password: "secret",
	}

	assert.Equal(t, "default", creds.Name)
	assert.Equal(t, "192.168.1.100", creds.Server)
	assert.Equal(t, 1433, creds.Port)
	assert.Equal(t, "PROTHEUS", creds.Database)
}
