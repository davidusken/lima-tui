package theme

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

	want := Dark
	want.Name = "mine"
	want.BG = tcell.ColorDefault
	want.FG = tcell.NewHexColor(0xc0caf5)
	want.SelectedBG = tcell.ColorPurple
	want.Running = tcell.ColorLime
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
	if th.BG != tcell.ColorDefault {
		t.Errorf("bg = %v, want default", th.BG)
	}
}

func TestParseSkinEmpty(t *testing.T) {
	th, err := parseSkin("empty", nil)
	if err != nil {
		t.Fatal(err)
	}
	if th.BG != Dark.BG || th.Name != "empty" {
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

func themeNames(themes []Theme) string {
	var names []string
	for _, t := range themes {
		names = append(names, t.Name)
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

	themes, idx, warnings, err := Load(dir, SkinRef{})
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

	_, idx, _, err = Load(dir, SkinRef{name: "zed"})
	if err != nil || idx != 3 {
		t.Errorf("want zed at 3, got %d, %v", idx, err)
	}
	_, idx, _, err = Load(dir, SkinRef{name: "light"})
	if err != nil || idx != 1 {
		t.Errorf("want light at 1, got %d, %v", idx, err)
	}

	if _, _, _, err := Load(dir, SkinRef{name: "missing"}); err == nil {
		t.Error("missing skin: want error")
	}
	if _, _, _, err := Load(dir, SkinRef{name: "broken"}); err == nil || !strings.Contains(err.Error(), "fg") {
		t.Errorf("asked-for broken skin: want its error, got %v", err)
	}
}

func TestLoadThemesByPath(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(t.TempDir(), "elsewhere.yaml")
	writeFile(t, other, "bg: default\n")

	themes, idx, _, err := Load(dir, SkinRef{path: other})
	if err != nil {
		t.Fatal(err)
	}
	if themes[idx].Name != "elsewhere" || themes[idx].BG != tcell.ColorDefault {
		t.Errorf("got %+v", themes[idx])
	}
	if got := themeNames(themes); got != "dark,light,elsewhere" {
		t.Errorf("themes = %s", got)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil || cfg.Skin != "" {
		t.Fatalf("missing config: got %+v, %v", cfg, err)
	}

	writeFile(t, filepath.Join(dir, "config.yaml"), "skin: solarized-dark\n")
	cfg, err = LoadConfig(dir)
	if err != nil || cfg.Skin != "solarized-dark" {
		t.Errorf("got %+v, %v", cfg, err)
	}

	writeFile(t, filepath.Join(dir, "config.yaml"), "skinn: x\n")
	if _, err := LoadConfig(dir); err == nil {
		t.Error("typo in config: want error")
	}
}

func TestConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got := ConfigDir(); got != filepath.Join("/xdg", "lima-tui") {
		t.Errorf("ConfigDir() = %s", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := ConfigDir(); !strings.HasSuffix(got, filepath.Join(".config", "lima-tui")) {
		t.Errorf("ConfigDir() = %s", got)
	}
}

// The example skins in the repo must load cleanly.
func TestExampleSkins(t *testing.T) {
	paths, err := filepath.Glob("../../skins/*.yaml")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no example skins found: %v", err)
	}
	for _, p := range paths {
		if _, err := loadSkin(p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestFlagSkin(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	writeFile(t, filepath.Join(cwd, "here.yaml"), "")

	tests := map[string]SkinRef{
		"":               {},
		"solarized-dark": {name: "solarized-dark"},
		"here.yaml":      {path: "here.yaml"},                                 // in the current directory
		"mine.yaml":      {path: filepath.Join("/cfg", "skins", "mine.yaml")}, // not here: skins folder
		"./mine.yaml":    {path: "./mine.yaml"},                               // explicit path stays
		"/abs/foo.yaml":  {path: "/abs/foo.yaml"},
	}
	for in, want := range tests {
		if got := FlagSkin("/cfg", in); got != want {
			t.Errorf("FlagSkin(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestConfigSkin(t *testing.T) {
	home := userHome()
	tests := map[string]SkinRef{
		"":                 {},
		"solarized-dark":   {name: "solarized-dark"},
		"foo.yaml":         {path: filepath.Join("/cfg", "skins", "foo.yaml")},
		"sub/foo.yaml":     {path: filepath.Join("/cfg", "sub", "foo.yaml")},
		"/abs/foo.yaml":    {path: "/abs/foo.yaml"},
		"~/skins/foo.yaml": {path: filepath.Join(home, "skins", "foo.yaml")},
	}
	for in, want := range tests {
		if got := ConfigSkin("/cfg", in); got != want {
			t.Errorf("ConfigSkin(%q) = %+v, want %+v", in, got, want)
		}
	}
}
