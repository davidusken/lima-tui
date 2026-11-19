// Command lima-tui is a terminal UI for managing Lima virtual machines.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rivo/tview"
)

// Set at build time by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

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
	if *skin == "" {
		*skin = configSkin(dir, cfg.Skin)
	}
	themes, idx, warnings, err := loadThemes(dir, *skin)
	if err != nil {
		fail(err)
	}
	var notice string
	if len(warnings) > 0 {
		notice = "[red]" + tview.Escape(strings.Join(warnings, "; ")) + "[-]"
	}

	if err := NewApp(themes, idx, notice).Run(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "lima-tui:", err)
	os.Exit(1)
}
