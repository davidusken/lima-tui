package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// config is ~/.config/lima-tui/config.yaml.
type config struct {
	Skin string `yaml:"skin"`
}

// skinRef is the skin the user asked for: a theme name, or a path to a skin
// file. At most one is set; neither means the default (dark).
type skinRef struct {
	name string
	path string
}

func userHome() string {
	home, _ := os.UserHomeDir() // empty on error; shortenHome then leaves paths alone
	return home
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

// looksLikePath tells a skin file ("foo.yaml", "dir/foo") from a theme name.
func looksLikePath(skin string) bool {
	return isSkinFile(skin) || strings.ContainsRune(skin, os.PathSeparator)
}

// flagSkin resolves --skin. Paths are relative to the current directory,
// except that a bare file name that isn't there means the file of that name
// in dir/skins.
func flagSkin(dir, skin string) skinRef {
	if !looksLikePath(skin) {
		return skinRef{name: skin}
	}
	if !strings.ContainsRune(skin, os.PathSeparator) {
		if _, err := os.Stat(skin); errors.Is(err, os.ErrNotExist) {
			return skinRef{path: filepath.Join(dir, "skins", skin)}
		}
	}
	return skinRef{path: skin}
}

// configSkin resolves the skin value from config.yaml in dir. Relative paths
// are relative to dir, not to wherever lima-tui was started: "foo.yaml" is
// dir/skins/foo.yaml, "sub/foo.yaml" is dir/sub/foo.yaml. A leading ~/ is the
// home directory, since YAML doesn't expand it.
func configSkin(dir, skin string) skinRef {
	switch {
	case !looksLikePath(skin):
		return skinRef{name: skin}
	case strings.HasPrefix(skin, "~"+string(os.PathSeparator)):
		return skinRef{path: filepath.Join(userHome(), skin[2:])}
	case filepath.IsAbs(skin):
		return skinRef{path: skin}
	case !strings.ContainsRune(skin, os.PathSeparator):
		return skinRef{path: filepath.Join(dir, "skins", skin)}
	default:
		return skinRef{path: filepath.Join(dir, skin)}
	}
}
