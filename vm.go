package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// VM represents a Lima VM
type VM struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	SSHAddress   string `json:"sshAddress"`
	VMType       string `json:"vmType"`
	Arch         string `json:"arch"`
	CPUs         int    `json:"cpus"`
	Memory       int64  `json:"memory"`
	Disk         int64  `json:"disk"`
	Dir          string `json:"dir"`
	SSHLocalPort int    `json:"sshLocalPort"`
}

// LoadVMs loads all Lima VMs using limactl
func LoadVMs() ([]VM, error) {
	cmd := exec.Command("limactl", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error loading VMs: %v", err)
	}

	// Parse JSONL format (one JSON object per line)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	vms := make([]VM, 0, len(lines))
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var vm VM
		if err := json.Unmarshal([]byte(line), &vm); err != nil {
			return nil, fmt.Errorf("error parsing VM data: %v", err)
		}
		vms = append(vms, vm)
	}

	return vms, nil
}

// FormatMemory formats memory size for display
func (v VM) FormatMemory() string {
	if v.Memory < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(v.Memory)/(1024*1024))
	}
	return fmt.Sprintf("%.1fGB", float64(v.Memory)/(1024*1024*1024))
}

// FormatDisk formats disk size for display
func (v VM) FormatDisk() string {
	if v.Disk < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(v.Disk)/(1024*1024))
	}
	return fmt.Sprintf("%.1fGB", float64(v.Disk)/(1024*1024*1024))
}

// GetStatusColor returns the color for the VM status
func (v VM) GetStatusColor() string {
	switch v.Status {
	case "Running":
		return "green"
	case "Stopped":
		return "red"
	case "Starting":
		return "yellow"
	case "Stopping":
		return "orange"
	default:
		return "gray"
	}
}
