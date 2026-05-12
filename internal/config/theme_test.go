package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTheme(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "memos-tui")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "theme.toml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write theme: %v", err)
	}
}

// --- validateColor ---

func TestValidateColor_empty(t *testing.T) {
	if err := validateColor("f", ""); err != nil {
		t.Errorf("empty should be valid, got %v", err)
	}
}

func TestValidateColor_ansi256(t *testing.T) {
	for _, s := range []string{"0", "1", "63", "255"} {
		if err := validateColor("f", s); err != nil {
			t.Errorf("%q should be valid ANSI 256, got %v", s, err)
		}
	}
}

func TestValidateColor_ansi256_outOfRange(t *testing.T) {
	for _, s := range []string{"256", "999", "-1"} {
		if err := validateColor("f", s); err == nil {
			t.Errorf("%q should be invalid (out of 0–255 range)", s)
		}
	}
}

func TestValidateColor_hexShort(t *testing.T) {
	if err := validateColor("f", "#ABC"); err != nil {
		t.Errorf("#ABC should be valid, got %v", err)
	}
}

func TestValidateColor_hexFull(t *testing.T) {
	if err := validateColor("f", "#1a2b3c"); err != nil {
		t.Errorf("#1a2b3c should be valid, got %v", err)
	}
}

func TestValidateColor_hexUppercase(t *testing.T) {
	if err := validateColor("f", "#AABBCC"); err != nil {
		t.Errorf("#AABBCC should be valid, got %v", err)
	}
}

func TestValidateColor_hexBadLength(t *testing.T) {
	for _, s := range []string{"#AB", "#ABCD", "#ABCDE", "#ABCDEFG"} {
		if err := validateColor("f", s); err == nil {
			t.Errorf("%q should be invalid (wrong hex length)", s)
		}
	}
}

func TestValidateColor_hexNoHash(t *testing.T) {
	if err := validateColor("f", "AABBCC"); err == nil {
		t.Error("hex without # should be invalid")
	}
}

func TestValidateColor_arbitraryString(t *testing.T) {
	for _, s := range []string{"red", "blue", "primary", "auto"} {
		if err := validateColor("f", s); err == nil {
			t.Errorf("%q should be invalid", s)
		}
	}
}

// --- LoadTheme ---

func TestLoadTheme_noFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	theme, err := LoadTheme()
	if err != nil {
		t.Fatalf("missing theme file should return defaults, got error: %v", err)
	}
	def := DefaultTheme()
	if theme != def {
		t.Errorf("expected default theme, got %+v", theme)
	}
}

func TestLoadTheme_fullValidFile(t *testing.T) {
	writeTheme(t, `
primary  = "#FF0000"
muted    = "240"
selected = "#0F0"
date     = "200"
tag      = "#336699"
error    = "1"
pinned   = "#FFD700"
`)
	theme, err := LoadTheme()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.Primary != "#FF0000" {
		t.Errorf("Primary = %q, want #FF0000", theme.Primary)
	}
	if theme.Muted != "240" {
		t.Errorf("Muted = %q, want 240", theme.Muted)
	}
	if theme.Selected != "#0F0" {
		t.Errorf("Selected = %q, want #0F0", theme.Selected)
	}
	if theme.Tag != "#336699" {
		t.Errorf("Tag = %q, want #336699", theme.Tag)
	}
}

func TestLoadTheme_partialFile_fillsDefaults(t *testing.T) {
	writeTheme(t, `primary = "#FF0000"`)
	theme, err := LoadTheme()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.Primary != "#FF0000" {
		t.Errorf("Primary = %q, want #FF0000", theme.Primary)
	}
	def := DefaultTheme()
	if theme.Muted != def.Muted {
		t.Errorf("unset Muted should be default %q, got %q", def.Muted, theme.Muted)
	}
	if theme.Tag != def.Tag {
		t.Errorf("unset Tag should be default %q, got %q", def.Tag, theme.Tag)
	}
}

func TestLoadTheme_invalidTOML(t *testing.T) {
	writeTheme(t, `primary = [not valid toml`)
	_, err := LoadTheme()
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
}

func TestLoadTheme_invalidColor_hex(t *testing.T) {
	writeTheme(t, `primary = "#GGGGGG"`)
	_, err := LoadTheme()
	if err == nil {
		t.Fatal("expected error for invalid hex color, got nil")
	}
	if !strings.Contains(err.Error(), "primary") {
		t.Errorf("error should name the offending field, got: %v", err)
	}
}

func TestLoadTheme_invalidColor_outOfRange(t *testing.T) {
	writeTheme(t, `tag = "256"`)
	_, err := LoadTheme()
	if err == nil {
		t.Fatal("expected error for out-of-range ANSI color, got nil")
	}
	if !strings.Contains(err.Error(), "tag") {
		t.Errorf("error should name the offending field, got: %v", err)
	}
}

func TestLoadTheme_invalidColor_namedColor(t *testing.T) {
	writeTheme(t, `error = "red"`)
	_, err := LoadTheme()
	if err == nil {
		t.Fatal("expected error for named color string, got nil")
	}
}

func TestLoadTheme_allFieldsInvalid_reportsFirst(t *testing.T) {
	writeTheme(t, `
primary = "notacolor"
muted   = "alsonotacolor"
`)
	_, err := LoadTheme()
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- DefaultTheme ---

func TestDefaultTheme_allFieldsNonEmpty(t *testing.T) {
	def := DefaultTheme()
	fields := map[string]string{
		"Primary":  def.Primary,
		"Muted":    def.Muted,
		"Selected": def.Selected,
		"Date":     def.Date,
		"Tag":      def.Tag,
		"Error":    def.Error,
		"Pinned":   def.Pinned,
	}
	for name, val := range fields {
		if val == "" {
			t.Errorf("DefaultTheme().%s is empty", name)
		}
		if err := validateColor(name, val); err != nil {
			t.Errorf("DefaultTheme().%s = %q is not a valid color: %v", name, val, err)
		}
	}
}
