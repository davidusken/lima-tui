package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VM is one Lima instance as reported by `limactl list --format json`.
type VM struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	SSHAddress   string `json:"sshAddress"`
	SSHLocalPort int    `json:"sshLocalPort"`
	VMType       string `json:"vmType"`
	Arch         string `json:"arch"`
	CPUs         int    `json:"cpus"`
	Memory       int64  `json:"memory"`
	Disk         int64  `json:"disk"`
	Dir          string `json:"dir"`
}

// Statuses limactl reports that lima-tui acts on. (Lima has no in-between
// states like "Starting"; those are lima-tui's own, see actions.go.)
const (
	statusRunning = "Running"
	statusStopped = "Stopped"
)

// queryTimeout bounds `limactl list` and `limactl --version`. Only one list
// runs at a time, so a hung one would otherwise stop all refreshes.
const queryTimeout = 30 * time.Second

// LoadVMs returns all Lima instances.
func LoadVMs() ([]VM, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "limactl", "list", "--format", "json")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("limactl list: no answer after %s", queryTimeout)
	}
	if err != nil {
		return nil, limactlError("list", err, stderr.Bytes())
	}
	return parseVMs(bytes.NewReader(out))
}

// parseVMs decodes the stream of JSON objects (one per line) that limactl prints.
func parseVMs(r io.Reader) ([]VM, error) {
	dec := json.NewDecoder(r)
	var vms []VM
	for {
		var vm VM
		err := dec.Decode(&vm)
		if errors.Is(err, io.EOF) {
			return vms, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parsing limactl output: %w", err)
		}
		vms = append(vms, vm)
	}
}

// limactl runs `limactl <args>` and returns a readable error on failure.
func limactl(args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("limactl", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return limactlError(args[0], err, stderr.Bytes())
	}
	return nil
}

func limactlError(subcmd string, err error, stderr []byte) error {
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("limactl not found in PATH; is Lima installed?")
	}
	if msg := logMessage(lastLine(stderr)); msg != "" {
		return fmt.Errorf("limactl %s: %s", subcmd, msg)
	}
	return fmt.Errorf("limactl %s: %w", subcmd, err)
}

// logMessage pulls the msg and error fields out of a logrus text line
// (`time="…" level=fatal msg="…" error="…"`), as "msg: error". A line that
// has neither field is returned unchanged.
func logMessage(line string) string {
	fields := logfmtFields(line)
	msg, errText := fields["msg"], fields["error"]
	switch {
	case msg != "" && errText != "":
		return msg + ": " + errText
	case msg != "":
		return msg
	case errText != "":
		return errText
	default:
		return line
	}
}

// logfmtFields parses key=value pairs, where a value is either bare or a
// Go-quoted string. Parsing stops at the first thing that doesn't fit.
func logfmtFields(line string) map[string]string {
	fields := map[string]string{}
	rest := strings.TrimSpace(line)
	for rest != "" {
		key, after, ok := strings.Cut(rest, "=")
		if !ok || key == "" || strings.ContainsAny(key, " \"") {
			break
		}
		var val string
		if strings.HasPrefix(after, `"`) {
			quoted, err := strconv.QuotedPrefix(after)
			if err != nil {
				break
			}
			val, _ = strconv.Unquote(quoted) // QuotedPrefix guarantees it's valid
			after = after[len(quoted):]
		} else {
			val, after, _ = strings.Cut(after, " ")
		}
		fields[key] = val
		rest = strings.TrimLeft(after, " ")
	}
	return fields
}

// lastLine returns the last non-empty line of b, which is where limactl
// puts the actual error after its progress logging.
func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

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

func userHome() string {
	home, _ := os.UserHomeDir() // empty on error; shortenHome then leaves paths alone
	return home
}

// sshEndpoint is what the Port column shows: just the port for localhost, host:port otherwise.
func (v VM) sshEndpoint() string {
	if v.SSHLocalPort == 0 {
		return "-"
	}
	if v.SSHAddress == "" || v.SSHAddress == "127.0.0.1" {
		return fmt.Sprint(v.SSHLocalPort)
	}
	return fmt.Sprintf("%s:%d", v.SSHAddress, v.SSHLocalPort)
}

// LimaVersion returns the installed Lima version (e.g. "2.3.0"), or "" if it
// can't be determined.
func LimaVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "limactl", "--version").Output()
	if err != nil {
		return ""
	}
	return parseLimaVersion(string(out))
}

// parseLimaVersion takes the last word of `limactl --version` output
// ("limactl version 2.3.0").
func parseLimaVersion(out string) string {
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}
