package theme

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
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

// parseSkin builds a theme from skin YAML, starting from Dark.
func parseSkin(name string, data []byte) (Theme, error) {
	var sf skinFile
	if err := decodeYAML(bytes.NewReader(data), &sf); err != nil {
		return Theme{}, err
	}

	t := Dark
	t.Name = name
	fields := []struct {
		key string
		val string
		dst *tcell.Color
	}{
		{"bg", sf.BG, &t.BG},
		{"fg", sf.FG, &t.FG},
		{"dim", sf.Dim, &t.Dim},
		{"title", sf.Title, &t.Title},
		{"header.fg", sf.Header.FG, &t.HeaderFG},
		{"header.bg", sf.Header.BG, &t.HeaderBG},
		{"selected.fg", sf.Selected.FG, &t.SelectedFG},
		{"selected.bg", sf.Selected.BG, &t.SelectedBG},
		{"button.fg", sf.Button.FG, &t.ButtonFG},
		{"button.bg", sf.Button.BG, &t.ButtonBG},
		{"status.running", sf.Status.Running, &t.Running},
		{"status.stopped", sf.Status.Stopped, &t.Stopped},
		{"status.starting", sf.Status.Starting, &t.Starting},
		{"status.stopping", sf.Status.Stopping, &t.Stopping},
	}
	for _, f := range fields {
		if f.val == "" {
			continue
		}
		c, err := parseColor(f.val)
		if err != nil {
			return Theme{}, fmt.Errorf("%s: %w", f.key, err)
		}
		*f.dst = c
	}
	return t, nil
}

// loadSkin reads a skin file; the theme is named after the file. Errors
// don't include the path, so callers can name the file as they like.
func loadSkin(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	return parseSkin(skinName(path), data)
}

func skinName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// Load returns the built-in themes followed by every skin in
// dir/skins (os.ReadDir sorts them by name), and the index of want.
//
// A skin that is asked for but can't be loaded is an error; other broken
// skins are skipped and reported in warnings.
func Load(dir string, want SkinRef) (themes []Theme, idx int, warnings []string, err error) {
	themes = append(themes, Builtins()...)

	skinsDir := filepath.Join(dir, "skins")
	entries, err := os.ReadDir(skinsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !isSkinFile(e.Name()) {
			continue
		}
		path := filepath.Join(skinsDir, e.Name())
		t, err := loadSkin(path)
		switch {
		case err != nil && skinName(path) == want.name:
			return nil, 0, nil, fmt.Errorf("skin %s: %w", path, err)
		case err != nil:
			// Shown in the status bar, so keep it short.
			warnings = append(warnings, fmt.Sprintf("skin %s: %v", e.Name(), err))
		case themeIndex(themes, t.Name) >= 0:
			warnings = append(warnings, fmt.Sprintf("skin %s: name %q already taken, skipped", e.Name(), t.Name))
		default:
			themes = append(themes, t)
		}
	}

	switch {
	case want.path != "":
		// A skin file given by path replaces a loaded one of the same name.
		t, err := loadSkin(want.path)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("skin %s: %w", want.path, err)
		}
		if i := themeIndex(themes, t.Name); i >= 0 {
			themes[i] = t
			return themes, i, warnings, nil
		}
		return append(themes, t), len(themes), warnings, nil
	case want.name == "":
		return themes, 0, warnings, nil
	}
	if i := themeIndex(themes, want.name); i >= 0 {
		return themes, i, warnings, nil
	}
	return nil, 0, nil, fmt.Errorf("skin %q not found (looked for built-ins dark, light and %s.yaml or .yml)",
		want.name, filepath.Join(skinsDir, want.name))
}

func isSkinFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

func themeIndex(themes []Theme, name string) int {
	for i, t := range themes {
		if t.Name == name {
			return i
		}
	}
	return -1
}
