// Package config reads and writes the shared QDivZero credentials file
// (~/.qdivzero/credentials), compatible with the qdivzero SDKs.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Credentials mirrors the JSON fields of ~/.qdivzero/credentials.
type Credentials struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Path returns the credentials file path.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: home dir: %w", err)
	}
	return filepath.Join(home, ".qdivzero", "credentials"), nil
}

// Read loads the credentials file. A missing file yields an empty Credentials
// with no error; a malformed file is an error.
func Read() (Credentials, error) {
	var c Credentials
	path, err := Path()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return c, fmt.Errorf("config: read: %w", err)
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return c, nil
}

// IsConfigured reports whether the credentials file carries an access token
// or email/password credentials.
func IsConfigured(c Credentials) bool {
	return c.AccessToken != "" || (c.Email != "" && c.Password != "")
}

// Write stores the credentials file (0600, dir 0700). Without force it
// refuses to overwrite an existing file.
func Write(c Credentials, force bool) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config: %s already exists (use --force to overwrite)", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("config: write: %w", err)
	}
	return nil
}
