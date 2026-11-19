// Command lima-tui is a terminal UI for managing Lima virtual machines.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// Set at build time by GoReleaser via -ldflags. Other builds fill them in
// from the build info in init.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	if version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		version, commit, date = versionFromBuildInfo(info, version, commit, date)
	}
}

// versionFromBuildInfo covers builds without GoReleaser's ldflags, such as
// `go install …@v1.2.0`: the module version and VCS stamp Go embeds in the
// binary replace whichever defaults they have a value for.
func versionFromBuildInfo(info *debug.BuildInfo, version, commit, date string) (string, string, string) {
	if v := info.Main.Version; v != "" && v != "(devel)" {
		version = v
	}
	for _, s := range info.Settings {
		switch {
		case s.Key == "vcs.revision" && s.Value != "":
			commit = s.Value[:min(7, len(s.Value))]
		case s.Key == "vcs.time" && s.Value != "":
			date = s.Value
		}
	}
	return version, commit, date
}

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	skin := flag.String("skin", "", "theme to start with: dark, light, a skin name from "+
		filepath.Join(configDir(), "skins")+", or a path to a skin file")
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage: lima-tui [--skin name] [--version]\n\n"+
			"A terminal UI for Lima VMs. Press ? inside the app for key bindings.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("lima-tui %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if _, err := exec.LookPath("limactl"); err != nil {
		fmt.Fprintln(os.Stderr, "lima-tui: limactl not found in PATH. Install Lima first: https://lima-vm.io/docs/installation/")
		os.Exit(1)
	}

	dir := configDir()
	cfg, err := loadConfig(dir)
	if err != nil {
		fail(err)
	}
	want := configSkin(dir, cfg.Skin) // --skin wins over config.yaml
	if *skin != "" {
		want = flagSkin(dir, *skin)
	}
	themes, idx, warnings, err := loadThemes(dir, want)
	if err != nil {
		fail(err)
	}
	var notice string
	if len(warnings) > 0 {
		notice = colored(themes[idx].stopped, strings.Join(warnings, "; "))
	}

	if err := newApp(themes, idx, notice).run(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "lima-tui:", err)
	os.Exit(1)
}
