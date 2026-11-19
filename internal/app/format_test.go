package app

import (
	"testing"

	"github.com/davidusken/lima-tui/internal/lima"
)

func TestHumanBytes(t *testing.T) {
	tests := map[int64]string{
		512 << 20:     "512MiB",
		4 << 30:       "4GiB",
		3 << 29:       "1.5GiB",
		100 << 30:     "100GiB",
		0:             "0MiB",
		(1 << 30) - 1: "1023MiB",
	}
	for in, want := range tests {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestShortenHome(t *testing.T) {
	tests := []struct{ path, home, want string }{
		{"/Users/me/.lima/default", "/Users/me", "~/.lima/default"},
		{"/home/me/.lima/default", "/home/me", "~/.lima/default"},
		{"/Users/me", "/Users/me", "~"},
		{"/Users/meg/.lima/x", "/Users/me", "/Users/meg/.lima/x"},
		{"/opt/lima/x", "", "/opt/lima/x"},
	}
	for _, tt := range tests {
		if got := shortenHome(tt.path, tt.home); got != tt.want {
			t.Errorf("shortenHome(%q, %q) = %q, want %q", tt.path, tt.home, got, tt.want)
		}
	}
}

func TestSSHEndpoint(t *testing.T) {
	tests := []struct {
		vm   lima.Instance
		want string
	}{
		{lima.Instance{SSHLocalPort: 60022}, "60022"},
		{lima.Instance{SSHAddress: "127.0.0.1", SSHLocalPort: 60022}, "60022"},
		{lima.Instance{SSHAddress: "192.168.5.15", SSHLocalPort: 22}, "192.168.5.15:22"},
		{lima.Instance{}, "-"},
	}
	for _, tt := range tests {
		if got := sshEndpoint(tt.vm); got != tt.want {
			t.Errorf("sshEndpoint(%+v) = %q, want %q", tt.vm, got, tt.want)
		}
	}
}
