package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig writes a TOML config to a temp dir and sets XDG_CONFIG_HOME so
// ConfigPath() and Load() resolve to it.
func writeConfig(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "memos-tui")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestLoad_success(t *testing.T) {
	writeConfig(t, `
default_profile = "home"

[profiles.home]
url   = "https://memos.example.com"
token = "memos_pat_abc123"
`)
	p, _, err := Load("home")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.URL != "https://memos.example.com" {
		t.Errorf("URL = %q, want %q", p.URL, "https://memos.example.com")
	}
	if p.Token != "memos_pat_abc123" {
		t.Errorf("Token = %q, want %q", p.Token, "memos_pat_abc123")
	}
}

func TestLoad_defaultProfile(t *testing.T) {
	writeConfig(t, `
default_profile = "work"

[profiles.work]
url   = "https://work.example.com"
token = "memos_pat_work"
`)
	// empty profileName should fall back to default_profile
	p, _, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.URL != "https://work.example.com" {
		t.Errorf("URL = %q, want %q", p.URL, "https://work.example.com")
	}
}

func TestLoad_multipleProfiles(t *testing.T) {
	writeConfig(t, `
default_profile = "home"

[profiles.home]
url   = "https://home.example.com"
token = "tok_home"

[profiles.work]
url   = "https://work.example.com"
token = "tok_work"
`)
	p, _, err := Load("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.URL != "https://work.example.com" {
		t.Errorf("URL = %q, want %q", p.URL, "https://work.example.com")
	}
}

func TestLoad_missingFile(t *testing.T) {
	// point XDG_CONFIG_HOME at an empty temp dir so no config.toml exists
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, _, err := Load("home")
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}

func TestLoad_missingProfile(t *testing.T) {
	writeConfig(t, `
default_profile = "home"

[profiles.home]
url   = "https://memos.example.com"
token = "tok"
`)
	_, _, err := Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
}

func TestLoad_emptyURL(t *testing.T) {
	writeConfig(t, `
default_profile = "bad"

[profiles.bad]
url   = ""
token = "tok"
`)
	_, _, err := Load("bad")
	if err == nil {
		t.Fatal("expected error for empty URL, got nil")
	}
}

func TestLoad_emptyToken(t *testing.T) {
	writeConfig(t, `
default_profile = "bad"

[profiles.bad]
url   = "https://memos.example.com"
token = ""
`)
	_, _, err := Load("bad")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestLoad_noDefaultProfile(t *testing.T) {
	writeConfig(t, `
[profiles.home]
url   = "https://memos.example.com"
token = "tok"
`)
	_, _, err := Load("")
	if err == nil {
		t.Fatal("expected error when no default_profile and no profile arg, got nil")
	}
}
