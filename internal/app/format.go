package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/davidusken/lima-tui/internal/lima"
)

// humanBytes formats a byte count the same way `limactl list` does (e.g. 4GiB, 512MiB).
func humanBytes(n int64) string {
	const (
		mib = 1 << 20
		gib = 1 << 30
	)
	switch {
	case n >= gib && n%gib == 0:
		return fmt.Sprintf("%dGiB", n/gib)
	case n >= gib:
		return fmt.Sprintf("%.1fGiB", float64(n)/gib)
	default:
		return fmt.Sprintf("%dMiB", n/mib)
	}
}

// shortenHome replaces the user's home directory prefix with ~.
func shortenHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}

// sshEndpoint is what the SSH column shows: just the port for localhost, host:port otherwise.
func sshEndpoint(v lima.Instance) string {
	if v.SSHLocalPort == 0 {
		return "-"
	}
	if v.SSHAddress == "" || v.SSHAddress == "127.0.0.1" {
		return fmt.Sprint(v.SSHLocalPort)
	}
	return fmt.Sprintf("%s:%d", v.SSHAddress, v.SSHLocalPort)
}
