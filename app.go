package main

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
)

// App represents the application state
type App struct {
	app         *tview.Application
	table       *tview.Table
	statusBar   *tview.TextView
	helpText    *tview.TextView
	vms         []VM
	lastUpdate  time.Time
	showHelp    bool
	refreshing  bool
	lightMode   bool  // true for light mode, false for dark mode
	shouldExit  bool  // flag to indicate if application should exit
}

// NewApp creates a new application instance
func NewApp() *App {
	a := &App{
		app:       tview.NewApplication(),
		table:     tview.NewTable(),
		statusBar: tview.NewTextView(),
		helpText:  tview.NewTextView(),
		lightMode: true, // Default to light mode for iTerm2
		shouldExit: false,
	}
	
	a.setupUI()
	a.setupKeybindings()
	// Load VMs after UI is set up but before app starts
	a.LoadVMs()
	
	return a
}

// setupKeybindings sets up keyboard event handlers
func (a *App) setupKeybindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		case tcell.KeyEsc:
			if a.showHelp {
				a.toggleHelp()
				return nil
			}
			a.Stop()
			return nil
		case tcell.KeyEnter:
			a.connectToSelected()
			return nil
		case tcell.KeyCtrlS:
			a.toggleVMState()
			return nil
		case tcell.KeyCtrlR:
			a.restartSelected()
			return nil
		case tcell.KeyCtrlD:
			a.deleteSelected()
			return nil
		case tcell.KeyCtrlT:
			a.ToggleTheme()
			return nil
		}
		
		switch event.Rune() {
		case 'q':
			a.Stop()
			return nil
		case 'h', '?':
			a.toggleHelp()
			return nil
		case 'r':
			a.Refresh()
			return nil
		case 't':
			a.ToggleTheme()
			return nil
		case 's':
			a.toggleVMState()
			return nil
		case 'd':
			a.deleteSelected()
			return nil
		case 'c':
			a.connectToSelected()
			return nil
		}
		
		return event
	})
}

// Run starts the application
func (a *App) Run() error {
	return a.app.Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.shouldExit = true
	a.app.Stop()
}

// ShouldExit returns whether the application should exit
func (a *App) ShouldExit() bool {
	return a.shouldExit
}

// UpdateStatus updates the status bar with a message
func (a *App) UpdateStatus(message string) {
	a.statusBar.SetText(fmt.Sprintf(" %s | Enter=Connect | Ctrl+S=Stop/Start | Ctrl+R=Restart | Ctrl+D=Delete | Ctrl+T=Theme | h=Help | q=Quit", message))
}

// LoadVMs loads VMs and updates the table
func (a *App) LoadVMs() {
	vms, err := LoadVMs()
	if err != nil {
		a.UpdateStatus(fmt.Sprintf("[red]%v", err))
		return
	}
	
	a.vms = vms
	a.updateTable()
	a.lastUpdate = time.Now()
	a.UpdateStatus(fmt.Sprintf("Loaded %d VMs - Last updated: %s", 
		len(a.vms), a.lastUpdate.Format("15:04:05")))
}

// GetSelectedVM returns the currently selected VM
func (a *App) GetSelectedVM() *VM {
	row, _ := a.table.GetSelection()
	if row < 1 || row > len(a.vms) {
		return nil
	}
	return &a.vms[row-1]
}

// Refresh refreshes the VM list
func (a *App) Refresh() {
	if a.refreshing {
		return
	}
	a.refreshing = true
	
	go func() {
		defer func() {
			a.refreshing = false
		}()
		
		a.app.QueueUpdateDraw(func() {
			a.UpdateStatus("Refreshing...")
		})
		
		time.Sleep(100 * time.Millisecond)
		
		a.app.QueueUpdateDraw(func() {
			a.LoadVMs()
		})
	}()
}
