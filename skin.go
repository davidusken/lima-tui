package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"go.yaml.in/yaml/v3"
)

// skinFile is the YAML layout of a skin. Every key is optional; anything left
// out keeps the built-in dark theme's colour.
type skinFile struct {
	BG       string  `yaml:"bg"`
	FG       string  `yaml:"fg"`
	Dim      string  `yaml:"dim"`
	Title    string  `yaml:"title"`
	Header   fgBg    `yaml:"header"`
	Selected fgBg    `yaml:"selected"`
	Button   fgBg    `yaml:"button"`
	Status   statusC `yaml:"status"`
}

type fgBg struct {
	FG string `yaml:"fg"`
	BG string `yaml:"bg"`
}

type statusC struct {
	Running  string `yaml:"running"`
	Stopped  string `yaml:"stopped"`
	Starting string `yaml:"starting"`
	Stopping string `yaml:"stopping"`
}

// config is ~/.config/lima-tui/config.yaml.
type config struct {
	Skin string `yaml:"skin"`
}

// parseColor turns a skin colour value into a tcell colour. It accepts W3C
// colour names ("aqua", "darkslategray"), "#rrggbb", and "default" or "-" for
// the terminal's own colour.
func parseColor(s string) (tcell.Color, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "default" || s == "-" {
		return tcell.ColorDefault, nil
	}
	// GetColor returns ColorDefault for anything it doesn't recognise.
	if c := tcell.GetColor(s); c != tcell.ColorDefault {
		return c, nil
	}
	return tcell.ColorDefault, fmt.Errorf("unknown colour %q (use a name, #rrggbb or default)", s)
}

// decodeYAML decodes r into v, rejecting unknown keys so typos are reported.
// An empty document is fine.
func decodeYAML(r io.Reader, v any) error {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// parseSkin builds a theme from skin YAML, starting from darkTheme.
func parseSkin(name string, data []byte) (theme, error) {
	var sf skinFile
	if err := decodeYAML(bytes.NewReader(data), &sf); err != nil {
		return theme{}, err
	}

	t := darkTheme
	t.name = name
	fields := []struct {
		key string
		val string
		dst *tcell.Color
	}{
		{"bg", sf.BG, &t.bg},
		{"fg", sf.FG, &t.fg},
		{"dim", sf.Dim, &t.dim},
		{"title", sf.Title, &t.title},
		{"header.fg", sf.Header.FG, &t.headerFg},
		{"header.bg", sf.Header.BG, &t.headerBg},
		{"selected.fg", sf.Selected.FG, &t.selectedFg},
		{"selected.bg", sf.Selected.BG, &t.selectedBg},
		{"button.fg", sf.Button.FG, &t.buttonFg},
		{"button.bg", sf.Button.BG, &t.buttonBg},
		{"status.running", sf.Status.Running, &t.running},
		{"status.stopped", sf.Status.Stopped, &t.stopped},
		{"status.starting", sf.Status.Starting, &t.starting},
		{"status.stopping", sf.Status.Stopping, &t.stopping},
	}
	for _, f := range fields {
		if f.val == "" {
			continue
		}
		c, err := parseColor(f.val)
		if err != nil {
			return theme{}, fmt.Errorf("%s: %w", f.key, err)
		}
		*f.dst = c
	}
	return t, nil
}

// loadSkin reads a skin file; the theme is named after the file. Errors
// don't include the path, so callers can name the file as they like.
func loadSkin(path string) (theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return theme{}, err
	}
	return parseSkin(skinName(path), data)
}

func skinName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// configDir is $XDG_CONFIG_HOME/lima-tui, or ~/.config/lima-tui. (Not
// os.UserConfigDir, which is ~/Library/Application Support on macOS.)
func configDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "lima-tui")
	}
	return filepath.Join(userHome(), ".config", "lima-tui")
}

// loadConfig reads config.yaml from dir. A missing file is not an error.
func loadConfig(dir string) (config, error) {
	var cfg config
	f, err := os.Open(filepath.Join(dir, "config.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	defer func() { _ = f.Close() }()
	if err := decodeYAML(f, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", f.Name(), err)
	}
	return cfg, nil
}

// configSkin resolves the skin value from config.yaml in dir. Unlike --skin,
// a relative path there is relative to dir, not to wherever lima-tui was
// started: "foo.yaml" is dir/skins/foo.yaml, "sub/foo.yaml" is dir/sub/foo.yaml.
// A leading ~/ is the home directory, since YAML doesn't expand it.
func configSkin(dir, skin string) string {
	switch {
	case !isSkinFile(skin) && !strings.ContainsRune(skin, os.PathSeparator):
		return skin // a skin name
	case strings.HasPrefix(skin, "~"+string(os.PathSeparator)):
		return filepath.Join(userHome(), skin[2:])
	case filepath.IsAbs(skin):
		return skin
	case !strings.ContainsRune(skin, os.PathSeparator):
		return filepath.Join(dir, "skins", skin)
	default:
		return filepath.Join(dir, skin)
	}
}

func isSkinFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

// loadThemes returns the built-in themes followed by every skin in
// dir/skins (os.ReadDir sorts them by name), and the index of the theme
// called want.
//
// want may be a built-in name, a skin name, or a path to a skin file (which
// is added to the list if it isn't in the skins folder). A bare file name
// ("foo.yaml") is looked for in the current directory, then the skins folder.
// Empty means "dark".
// A skin that is asked for but can't be loaded is an error; other broken
// skins are skipped and reported in warnings.
func loadThemes(dir, want string) (themes []theme, idx int, warnings []string, err error) {
	themes = append(themes, builtinThemes...)

	skinsDir := filepath.Join(dir, "skins")
	entries, err := os.ReadDir(skinsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil, err
	}

	var wantErr error
	wantName := want
	if isSkinFile(want) || strings.ContainsRune(want, os.PathSeparator) {
		wantName = ""
	}
	for _, e := range entries {
		if e.IsDir() || !isSkinFile(e.Name()) {
			continue
		}
		path := filepath.Join(skinsDir, e.Name())
		t, err := loadSkin(path)
		switch {
		case err != nil && skinName(path) == wantName:
			wantErr = fmt.Errorf("skin %s: %w", path, err)
		case err != nil:
			// Shown in the status bar, so keep it short.
			warnings = append(warnings, fmt.Sprintf("skin %s: %v", e.Name(), err))
		case themeIndex(themes, t.name) >= 0:
			warnings = append(warnings, fmt.Sprintf("skin %s: name %q already taken, skipped", e.Name(), t.name))
		default:
			themes = append(themes, t)
		}
	}
	if wantErr != nil {
		return nil, 0, nil, wantErr
	}

	switch {
	case want == "":
		return themes, 0, warnings, nil
	case wantName == "":
		path := want
		// A bare file name that isn't in the current directory means the
		// file of that name in the skins folder.
		if !strings.ContainsRune(want, os.PathSeparator) {
			if _, err := os.Stat(want); errors.Is(err, os.ErrNotExist) {
				path = filepath.Join(skinsDir, want)
			}
		}
		t, err := loadSkin(path)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("skin %s: %w", want, err)
		}
		if i := themeIndex(themes, t.name); i >= 0 {
			themes[i] = t
			return themes, i, warnings, nil
		}
		themes = append(themes, t)
		return themes, len(themes) - 1, warnings, nil
	}
	if i := themeIndex(themes, want); i >= 0 {
		return themes, i, warnings, nil
	}
	return nil, 0, nil, fmt.Errorf("skin %q not found (looked for built-ins dark, light and %s)",
		want, filepath.Join(skinsDir, want+".yaml"))
}

func themeIndex(themes []theme, name string) int {
	for i, t := range themes {
		if t.name == name {
			return i
		}
	}
	return -1
}
