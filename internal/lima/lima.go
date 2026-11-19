// Package lima runs limactl and reads what it prints.
package lima

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Instance is one Lima VM as reported by `limactl list --format json`.
type Instance struct {
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
// states like "Starting"; those are lima-tui's own, see internal/app.)
const (
	StatusRunning = "Running"
	StatusStopped = "Stopped"
)

// queryTimeout bounds `limactl list` and `limactl --version`. Only one list
// runs at a time, so a hung one would otherwise stop all refreshes.
const queryTimeout = 30 * time.Second

// List returns all Lima instances.
func List() ([]Instance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "limactl", limactlArgs("list", "--format", "json")...)
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
func parseVMs(r io.Reader) ([]Instance, error) {
	dec := json.NewDecoder(r)
	var vms []Instance
	for {
		var vm Instance
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

// limactlArgs puts the global flags lima-tui always passes in front of args.
// JSON logs make the error on stderr easy to pick out (see logMessage).
func limactlArgs(args ...string) []string {
	return append([]string{"--log-format", "json"}, args...)
}

// Run runs `limactl <args>` and returns a readable error on failure.
func Run(args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("limactl", limactlArgs(args...)...)
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

// logMessage pulls the msg and error fields out of a limactl JSON log line
// (`{"level":"fatal","msg":"…","error":"…"}`), as "msg: error". Anything
// else, such as output from a command limactl ran, is returned unchanged.
func logMessage(line string) string {
	var entry struct{ Msg, Error string }
	if json.Unmarshal([]byte(line), &entry) != nil {
		return line
	}
	switch {
	case entry.Msg != "" && entry.Error != "":
		return entry.Msg + ": " + entry.Error
	case entry.Msg != "":
		return entry.Msg
	case entry.Error != "":
		return entry.Error
	default:
		return line
	}
}

// lastLine returns the last non-empty line of b, which is where limactl
// puts the actual error after its progress logging.
func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// Version returns the installed Lima version (e.g. "2.3.0"), or "" if it
// can't be determined.
func Version() string {
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
