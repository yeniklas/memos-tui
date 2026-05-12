package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Theme holds the color values used by the TUI. Each field is either empty
// (inherit the default) or a valid lipgloss color: an ANSI 256 number (0–255)
// or a hex color (#RGB or #RRGGBB).
type Theme struct {
	Primary  string `toml:"primary"`
	Muted    string `toml:"muted"`
	Selected string `toml:"selected"`
	Date     string `toml:"date"`
	Tag      string `toml:"tag"`
	Error    string `toml:"error"`
	Pinned   string `toml:"pinned"`
}

func DefaultTheme() Theme {
	return Theme{
		Primary:  "63",
		Muted:    "241",
		Selected: "212",
		Date:     "246",
		Tag:      "79",
		Error:    "160",
		Pinned:   "220",
	}
}

var hexColorRE = regexp.MustCompile(`^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)

// validateColor returns an error if s is not a valid color value.
// Empty string is valid (means: use default).
func validateColor(field, s string) error {
	if s == "" {
		return nil
	}
	if hexColorRE.MatchString(s) {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err == nil && n >= 0 && n <= 255 {
		return nil
	}
	return fmt.Errorf("theme field %q: %q is not a valid color (expected empty, #RGB, #RRGGBB, or 0–255)", field, s)
}

func (t Theme) validate() error {
	fields := []struct {
		name  string
		value string
	}{
		{"primary", t.Primary},
		{"muted", t.Muted},
		{"selected", t.Selected},
		{"date", t.Date},
		{"tag", t.Tag},
		{"error", t.Error},
		{"pinned", t.Pinned},
	}
	for _, f := range fields {
		if err := validateColor(f.name, f.value); err != nil {
			return err
		}
	}
	return nil
}

// fillDefaults replaces any empty field with the corresponding default color.
func fillDefaults(t *Theme) {
	def := DefaultTheme()
	if t.Primary == "" {
		t.Primary = def.Primary
	}
	if t.Muted == "" {
		t.Muted = def.Muted
	}
	if t.Selected == "" {
		t.Selected = def.Selected
	}
	if t.Date == "" {
		t.Date = def.Date
	}
	if t.Tag == "" {
		t.Tag = def.Tag
	}
	if t.Error == "" {
		t.Error = def.Error
	}
	if t.Pinned == "" {
		t.Pinned = def.Pinned
	}
}

func ThemePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "memos-tui", "theme.toml"), nil
}

// LoadTheme loads the theme from ~/.config/memos-tui/theme.toml.
// If the file does not exist, the default theme is returned silently.
// If the file exists but is invalid (bad TOML or invalid color), an error is returned.
func LoadTheme() (Theme, error) {
	path, err := ThemePath()
	if err != nil {
		return DefaultTheme(), nil
	}

	var t Theme
	if _, err := toml.DecodeFile(path, &t); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultTheme(), nil
		}
		return Theme{}, fmt.Errorf("theme: %w", err)
	}

	if err := t.validate(); err != nil {
		return Theme{}, err
	}

	fillDefaults(&t)
	return t, nil
}
