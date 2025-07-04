package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// setupUI initializes the UI components
func (a *App) setupUI() {
	// Set the theme first
	a.applyTheme()
	
	// Create table with k9s-style configuration
	a.table = tview.NewTable()
	a.table.SetBorder(false)
	a.table.SetTitle("  LIMA VM MANAGER  ")
	a.table.SetTitleAlign(tview.AlignCenter)
	
	// This is the key setting from k9s: only rows selectable, not individual cells
	a.table.SetSelectable(true, false)
	a.table.SetFixed(1, 0)
	
	// Apply theme-specific table styling
	a.applyTableTheme()
	
	// Create status bar
	a.statusBar = tview.NewTextView()
	a.statusBar.SetBorder(false)
	a.statusBar.SetDynamicColors(true)
	a.statusBar.SetTextAlign(tview.AlignLeft)
	
	// Create help text
	a.helpText = tview.NewTextView()
	a.helpText.SetBorder(false)
	a.helpText.SetTitle("  HELP  ")
	a.helpText.SetDynamicColors(true)
	a.helpText.SetText(getHelpText())
	
	// Apply theme to all components
	a.applyComponentTheme()
	
	// Setup layout
	a.setupLayout()
}

// SetupLayout sets up the UI layout (public method for external access)
func (a *App) SetupLayout() {
	a.setupLayout()
}

// setupLayout sets up the UI layout
func (a *App) setupLayout() {
	// Layout
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	
	if a.showHelp {
		mainFlex := tview.NewFlex()
		mainFlex.AddItem(a.table, 0, 2, true)
		mainFlex.AddItem(tview.NewBox(), 1, 0, false) // Spacer
		mainFlex.AddItem(a.helpText, 0, 1, false)
		flex.AddItem(mainFlex, 0, 1, true)
	} else {
		flex.AddItem(a.table, 0, 1, true)
	}
	
	flex.AddItem(tview.NewBox(), 1, 0, false) // Spacer
	flex.AddItem(a.statusBar, 1, 0, false)
	
	a.app.SetRoot(flex, true)
}

// updateTable updates the table with current VM data
func (a *App) updateTable() {
	a.table.Clear()
	
	// Set headers with theme-aware formatting
	headers := []string{"Name", "Status", "Port", "Type", "Arch", "CPUs", "Memory", "Disk", "Dir"}
	for col, header := range headers {
		cell := tview.NewTableCell(header)
		if a.lightMode {
			cell.SetTextColor(tcell.ColorWhite)
			cell.SetBackgroundColor(tcell.ColorDarkSlateGray)
		} else {
			cell.SetTextColor(tcell.ColorWhite)
			cell.SetBackgroundColor(tcell.ColorDarkGreen)
		}
		cell.SetAttributes(tcell.AttrBold)
		cell.SetSelectable(false)
		cell.SetExpansion(1)
		a.table.SetCell(0, col, cell)
	}
	
	// Add VM rows
	for i, vm := range a.vms {
		a.addVMRow(i+1, vm)
	}
	
	// Select first data row if available
	if len(a.vms) > 0 {
		a.table.Select(1, 0)
	}
	
	a.applyTableTheme()
}

// addVMRow adds a VM row to the table
func (a *App) addVMRow(row int, vm VM) {
	// Format data
	status := vm.Status
	sshAddress := fmt.Sprintf("%d", vm.SSHLocalPort)
	if vm.SSHAddress != "" && vm.SSHAddress != "127.0.0.1" {
		sshAddress = fmt.Sprintf("%s:%d", vm.SSHAddress, vm.SSHLocalPort)
	}
	
	memoryGB := float64(vm.Memory) / (1024 * 1024 * 1024)
	diskGB := float64(vm.Disk) / (1024 * 1024 * 1024)
	
	dirPath := vm.Dir
	if strings.HasPrefix(dirPath, "/Users/") {
		homeDir, _ := os.UserHomeDir()
		if strings.HasPrefix(dirPath, homeDir) {
			dirPath = strings.Replace(dirPath, homeDir, "~", 1)
		}
	}
	
	// Create cells
	cells := []string{
		vm.Name,
		status,
		sshAddress,
		vm.VMType,
		vm.Arch,
		strconv.Itoa(vm.CPUs),
		fmt.Sprintf("%.0fG", memoryGB),
		fmt.Sprintf("%.0fG", diskGB),
		dirPath,
	}
	
	for col, content := range cells {
		cell := tview.NewTableCell(content)
		cell.SetExpansion(1)
		
		// Color code status
		if col == 1 { // Status column
			switch vm.Status {
			case "Running":
				cell.SetTextColor(tcell.ColorGreen)
			case "Stopped":
				cell.SetTextColor(tcell.ColorRed)
			case "Starting", "Stopping":
				cell.SetTextColor(tcell.ColorYellow)
			default:
				if a.lightMode {
					cell.SetTextColor(tcell.ColorBlack)
				} else {
					cell.SetTextColor(tcell.ColorWhite)
				}
			}
		} else {
			// Regular text color based on theme
			if a.lightMode {
				cell.SetTextColor(tcell.ColorBlack)
			} else {
				cell.SetTextColor(tcell.ColorWhite)
			}
		}
		
		a.table.SetCell(row, col, cell)
	}
}

// toggleHelp toggles the help panel
func (a *App) toggleHelp() {
	a.showHelp = !a.showHelp
	a.setupLayout()
}

// getHelpText returns the help text
func getHelpText() string {
	return `LIMA VM MANAGER

Navigation:
  Up/Down     - Select VM
  Page Up/Dn  - Scroll faster

Actions:
  Enter       - Connect to VM
  Ctrl+S      - Stop/Start VM
  Ctrl+R      - Restart VM
  Ctrl+D      - Delete VM
  r           - Refresh list

View:
  h, ?        - Toggle help
  Ctrl+T, t   - Toggle light/dark theme

Other:
  q, Esc      - Quit
  Ctrl+C      - Force quit

Status colors:
  Running     - Green
  Stopped     - Red
  Starting    - Yellow
  Stopping    - Orange

TIP: Use Ctrl+S to toggle VM state,
Ctrl+R to restart, and Ctrl+D to delete.
Connect with Enter when running.`
}
