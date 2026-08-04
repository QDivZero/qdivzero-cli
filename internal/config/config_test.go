package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".qdivzero")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if content != "" {
		if err := os.WriteFile(filepath.Join(dir, "credentials"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func TestReadMissingFileReturnsEmpty(t *testing.T) {
	writeConfig(t, "")
	c, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if c.AccessToken != "" || c.Email != "" {
		t.Fatalf("expected empty config, got %+v", c)
	}
}

func TestReadParsesCredentials(t *testing.T) {
	writeConfig(t, `{"email":"a@b.c","password":"p","access_token":"t1","refresh_token":"r1"}`)
	c, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if c.Email != "a@b.c" || c.Password != "p" || c.AccessToken != "t1" || c.RefreshToken != "r1" {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestReadMalformedReturnsError(t *testing.T) {
	writeConfig(t, "not json")
	if _, err := Read(); err == nil {
		t.Fatal("Read: expected error for malformed file")
	}
}

func TestWriteCreatesFileWithMode(t *testing.T) {
	home := writeConfig(t, "")
	c := Credentials{AccessToken: "tok"}
	if err := Write(c, false); err != nil {
		t.Fatalf("Write: %v", err)
	}
	path := filepath.Join(home, ".qdivzero", "credentials")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestWriteRefusesOverwriteWithoutForce(t *testing.T) {
	writeConfig(t, `{"access_token":"old"}`)
	if err := Write(Credentials{AccessToken: "new"}, false); err == nil {
		t.Fatal("Write: expected error without force")
	}
	if err := Write(Credentials{AccessToken: "new"}, true); err != nil {
		t.Fatalf("Write force: %v", err)
	}
}
