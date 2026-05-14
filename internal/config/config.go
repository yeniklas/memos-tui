package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Profile struct {
	URL        string `toml:"url"`
	Token      string `toml:"token"`
	Markdown   *bool  `toml:"markdown"`    // nil → default true (enabled)
	JournalTags []string `toml:"journal_tags"` // tags applied to journal entries; defaults to ["diary"]
}

// MarkdownEnabled returns true unless the profile explicitly sets markdown = false.
func (p Profile) MarkdownEnabled() bool {
	if p.Markdown == nil {
		return true
	}
	return *p.Markdown
}

// JournalTagsOrDefault returns the configured journal tags, or ["diary"] if unset.
func (p Profile) JournalTagsOrDefault() []string {
	if len(p.JournalTags) == 0 {
		return []string{"diary"}
	}
	return p.JournalTags
}

type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	Profiles       map[string]Profile `toml:"profiles"`
}

func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "memos-tui", "config.toml"), nil
}

func Load(profileName string) (Profile, string, error) {
	path, err := ConfigPath()
	if err != nil {
		return Profile{}, "", err
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Profile{}, "", fmt.Errorf("config file not found at %s\n\nCreate it with:\n\n  [profiles.default]\n  url   = \"https://your-memos-host\"\n  token = \"memos_pat_...\"\n\n  default_profile = \"default\"", path)
		}
		return Profile{}, "", fmt.Errorf("parse config: %w", err)
	}

	if profileName == "" {
		profileName = cfg.DefaultProfile
	}
	if profileName == "" {
		return Profile{}, "", fmt.Errorf("no profile specified and no default_profile set in config")
	}

	p, ok := cfg.Profiles[profileName]
	if !ok {
		return Profile{}, "", fmt.Errorf("profile %q not found in config", profileName)
	}
	if p.URL == "" || p.Token == "" {
		return Profile{}, "", fmt.Errorf("profile %q must have both url and token set", profileName)
	}
	return p, profileName, nil
}
