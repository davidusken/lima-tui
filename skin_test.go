package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		in   string
		want tcell.Color
	}{
		{"aqua", tcell.ColorAqua},
		{"DarkSlateGray", tcell.ColorDarkSlateGray},
		{"#c0caf5", tcell.NewHexColor(0xc0caf5)},
		{"default", tcell.ColorDefault},
		{"-", tcell.ColorDefault},
		{" red ", tcell.ColorRed},
	}
	for _, tt := range tests {
		got, err := parseColor(tt.in)
		if err != nil {
			t.Errorf("parseColor(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseColor(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}

	for _, bad := range []string{"notacolour", "#12345", "#gggggg"} {
		if _, err := parseColor(bad); err == nil {
			t.Errorf("parseColor(%q): want error", bad)
		}
	}
}

func TestParseSkinFallsBackToDark(t *testing.T) {
	th, err := parseSkin("mine", []byte(`
bg: default
fg: "#c0caf5"
selected:
  bg: purple
status:
  running: lime
`))
	if err != nil {
		t.Fatal(err)
	}

	want := darkTheme
	want.name = "mine"
	want.bg = tcell.ColorDefault
	want.fg = tcell.NewHexColor(0xc0caf5)
	want.selectedBg = tcell.ColorPurple
	want.running = tcell.ColorLime
	if th != want {
		t.Errorf("got %+v\nwant %+v", th, want)
	}
}

// A bare - is a YAML list marker, so "-" must be quoted (the docs say so).
func TestParseSkinQuotedDash(t *testing.T) {
	th, err := parseSkin("x", []byte("bg: \"-\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if th.bg != tcell.ColorDefault {
		t.Errorf("bg = %v, want default", th.bg)
	}
}

func TestParseSkinEmpty(t *testing.T) {
	th, err := parseSkin("empty", nil)
	if err != nil {
		t.Fatal(err)
	}
	if th.bg != darkTheme.bg || th.name != "empty" {
		t.Errorf("empty skin should equal dark theme, got %+v", th)
	}
}

func TestParseSkinErrors(t *testing.T) {
	tests := map[string]string{
		"unknown colour": "fg: blurple\n",
		"unknown key":    "foreground: red\n",
		"nested key":     "status:\n  paused: red\n",
		"bad yaml":       "fg: [\n",
	}
	for name, in := range tests {
		if _, err := parseSkin("x", []byte(in)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}

	_, err := parseSkin("x", []byte("header:\n  bg: nope\n"))
	if err == nil || !strings.Contains(err.Error(), "header.bg") {
		t.Errorf("error should name the key, got %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func themeNames(themes []theme) string {
	var names []string
	for _, t := range themes {
		names = append(names, t.name)
	}
	return strings.Join(names, ",")
}

func TestLoadThemes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skins", "zed.yaml"), "fg: red\n")
	writeFile(t, filepath.Join(dir, "skins", "alpha.yml"), "fg: blue\n")
	writeFile(t, filepath.Join(dir, "skins", "broken.yaml"), "fg: nope\n")
	writeFile(t, filepath.Join(dir, "skins", "dark.yaml"), "fg: red\n")
	writeFile(t, filepath.Join(dir, "skins", "README.md"), "not a skin")

	themes, idx, warnings, err := loadThemes(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := themeNames(themes); got != "dark,light,alpha,zed" {
		t.Errorf("themes = %s", got)
	}
	if idx != 0 {
		t.Errorf("idx = %d, want 0 (dark)", idx)
	}
	if len(warnings) != 2 {
		t.Errorf("want warnings for broken.yaml and dark.yaml, got %q", warnings)
	}

	_, idx, _, err = loadThemes(dir, "zed")
	if err != nil || idx != 3 {
		t.Errorf("want zed at 3, got %d, %v", idx, err)
	}
	_, idx, _, err = loadThemes(dir, "light")
	if err != nil || idx != 1 {
		t.Errorf("want light at 1, got %d, %v", idx, err)
	}

	if _, _, _, err := loadThemes(dir, "missing"); err == nil {
		t.Error("missing skin: want error")
	}
	if _, _, _, err := loadThemes(dir, "broken"); err == nil || !strings.Contains(err.Error(), "fg") {
		t.Errorf("asked-for broken skin: want its error, got %v", err)
	}
}

func TestLoadThemesByPath(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(t.TempDir(), "elsewhere.yaml")
	writeFile(t, other, "bg: default\n")

	themes, idx, _, err := loadThemes(dir, other)
	if err != nil {
		t.Fatal(err)
	}
	if themes[idx].name != "elsewhere" || themes[idx].bg != tcell.ColorDefault {
		t.Errorf("got %+v", themes[idx])
	}
	if got := themeNames(themes); got != "dark,light,elsewhere" {
		t.Errorf("themes = %s", got)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := loadConfig(dir)
	if err != nil || cfg.Skin != "" {
		t.Fatalf("missing config: got %+v, %v", cfg, err)
	}

	writeFile(t, filepath.Join(dir, "config.yaml"), "skin: solarized-dark\n")
	cfg, err = loadConfig(dir)
	if err != nil || cfg.Skin != "solarized-dark" {
		t.Errorf("got %+v, %v", cfg, err)
	}

	writeFile(t, filepath.Join(dir, "config.yaml"), "skinn: x\n")
	if _, err := loadConfig(dir); err == nil {
		t.Error("typo in config: want error")
	}
}

func TestConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got := configDir(); got != filepath.Join("/xdg", "lima-tui") {
		t.Errorf("configDir() = %s", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := configDir(); !strings.HasSuffix(got, filepath.Join(".config", "lima-tui")) {
		t.Errorf("configDir() = %s", got)
	}
}

// The example skins in the repo must load cleanly.
func TestExampleSkins(t *testing.T) {
	paths, err := filepath.Glob("skins/*.yaml")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no example skins found: %v", err)
	}
	for _, p := range paths {
		if _, err := loadSkin(p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestColorTag(t *testing.T) {
	if got := colorTag(tcell.ColorDefault); got != "-" {
		t.Errorf("default = %q", got)
	}
	if got := colorTag(tcell.ColorGreen); got != "green" {
		t.Errorf("green = %q", got)
	}
	if got := colorTag(tcell.NewHexColor(0x2aa198)); got != "#2AA198" {
		t.Errorf("hex = %q", got)
	}
}

func TestLoadThemesBareFileName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skins", "mine.yaml"), "fg: red\n")
	t.Chdir(t.TempDir()) // no mine.yaml here

	themes, idx, _, err := loadThemes(dir, "mine.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if themes[idx].name != "mine" || themes[idx].fg != tcell.ColorRed {
		t.Errorf("got %+v", themes[idx])
	}
}

func TestConfigSkin(t *testing.T) {
	home := userHome()
	tests := map[string]string{
		"":                 "",
		"solarized-dark":   "solarized-dark",
		"foo.yaml":         filepath.Join("/cfg", "skins", "foo.yaml"),
		"sub/foo.yaml":     filepath.Join("/cfg", "sub", "foo.yaml"),
		"/abs/foo.yaml":    "/abs/foo.yaml",
		"~/skins/foo.yaml": filepath.Join(home, "skins", "foo.yaml"),
	}
	for in, want := range tests {
		if got := configSkin("/cfg", in); got != want {
			t.Errorf("configSkin(%q) = %q, want %q", in, got, want)
		}
	}
}
