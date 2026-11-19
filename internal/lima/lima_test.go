package lima

import (
	"strings"
	"testing"
)

func TestParseVMs(t *testing.T) {
	input := `{"name":"default","status":"Running","sshLocalPort":60022,"vmType":"vz","arch":"aarch64","cpus":4,"memory":4294967296,"disk":107374182400,"dir":"/Users/me/.lima/default"}
{"name":"docker","status":"Stopped","vmType":"qemu","arch":"x86_64","cpus":2,"memory":2147483648,"disk":10737418240,"dir":"/Users/me/.lima/docker"}
`
	vms, err := parseVMs(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(vms) != 2 {
		t.Fatalf("got %d VMs, want 2", len(vms))
	}
	if vms[0].Name != "default" || vms[0].Status != StatusRunning || vms[0].CPUs != 4 || vms[0].SSHLocalPort != 60022 {
		t.Errorf("first VM parsed wrong: %+v", vms[0])
	}
	if vms[1].Name != "docker" || vms[1].Status != StatusStopped {
		t.Errorf("second VM parsed wrong: %+v", vms[1])
	}
}

func TestParseVMsEmpty(t *testing.T) {
	vms, err := parseVMs(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(vms) != 0 {
		t.Fatalf("got %d VMs, want 0", len(vms))
	}
}

func TestParseVMsInvalid(t *testing.T) {
	if _, err := parseVMs(strings.NewReader("{not json")); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestLastLine(t *testing.T) {
	stderr := []byte("time=\"...\" level=info msg=\"Stopping\"\ntime=\"...\" level=fatal msg=\"instance not found\"\n\n")
	if got, want := lastLine(stderr), `time="..." level=fatal msg="instance not found"`; got != want {
		t.Errorf("lastLine = %q, want %q", got, want)
	}
	if got := lastLine(nil); got != "" {
		t.Errorf("lastLine(nil) = %q, want empty", got)
	}
}

func TestParseLimaVersion(t *testing.T) {
	tests := map[string]string{
		"limactl version 2.3.0\n":    "2.3.0",
		"limactl version 1.0.0-rc.1": "1.0.0-rc.1",
		"":                           "",
	}
	for in, want := range tests {
		if got := parseLimaVersion(in); got != want {
			t.Errorf("parseLimaVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLogMessage(t *testing.T) {
	tests := map[string]string{
		// Real limactl --log-format json output.
		`{"level":"fatal","msg":"open /Users/me/.lima/x/lima.yaml: no such file or directory","time":"2026-11-19T09:26:54+01:00"}`: "open /Users/me/.lima/x/lima.yaml: no such file or directory",
		`{"level":"fatal","msg":"failed to stop","error":"exit status 1"}`:                                                         "failed to stop: exit status 1",
		`{"level":"fatal","error":"exit status 1"}`:                                                                                "exit status 1",
		`{"level":"info"}`:      `{"level":"info"}`,
		"some plain error text": "some plain error text",
		"":                      "",
	}
	for in, want := range tests {
		if got := logMessage(in); got != want {
			t.Errorf("logMessage(%q) = %q, want %q", in, got, want)
		}
	}
}
