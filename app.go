package main

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rivo/tview"
)

// refreshInterval is how often the VM list is reloaded in the background.
const refreshInterval = 3 * time.Second

const (
	pageMain    = "main"
	pageConfirm = "confirm"
)

// app holds the UI and the last known VM list. Fields above loadMu are only
// touched from the tview event loop (directly or via QueueUpdateDraw).
type app struct {
	tv         *tview.Application
	pages      *tview.Pages
	root       *tview.Flex
	table      *tview.Table
	statusBar  *tview.TextView
	helpText   *tview.TextView
	header     *tview.Flex
	headerInfo *tview.Table
	headerKeys *tview.Table

	vms         []instance
	lastUpdate  time.Time
	message     string
	refreshing  bool // a reload the user asked for is running
	showHelp    bool
	showHeader  bool
	limaVersion string
	home        string
	themes      []theme
	themeIdx    int
	busy        map[string]string // VM name -> action in progress (actStarting, …)

	loadMu      sync.Mutex
	loading     bool // a reload is running
	reloadAgain bool // reload once more when it finishes

	// suspended is set while a VM shell has the terminal. The event loop is
	// blocked then, so background refreshes would only pile up.
	suspended atomic.Bool
}

// newApp builds the UI. themes must not be empty; idx is the theme to start
// with and notice, if set, is shown in the status bar.
func newApp(themes []theme, idx int, notice string) *app {
	a := &app{
		tv:         tview.NewApplication(),
		pages:      tview.NewPages(),
		root:       tview.NewFlex(),
		table:      tview.NewTable(),
		statusBar:  tview.NewTextView(),
		helpText:   tview.NewTextView(),
		header:     tview.NewFlex(),
		headerInfo: tview.NewTable(),
		headerKeys: tview.NewTable(),
		showHeader: true,
		home:       userHome(),
		themes:     themes,
		themeIdx:   idx,
		busy:       map[string]string{},
		message:    notice,
	}
	a.setupUI()
	a.tv.SetInputCapture(a.handleKey)
	return a
}

// run shows the UI until the user quits.
func (a *app) run() error {
	stop := make(chan struct{})
	defer close(stop)

	a.refresh()
	go func() {
		v := queryLimaVersion()
		a.tv.QueueUpdateDraw(func() {
			a.limaVersion = v
			a.renderHeader()
		})
	}()
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if !a.suspended.Load() {
					a.refresh()
				}
			case <-stop:
				return
			}
		}
	}()

	return a.tv.SetRoot(a.pages, true).Run()
}

func (a *app) quit() {
	a.tv.Stop()
}

// manualRefresh is the r key: reload now and say so until it's done.
func (a *app) manualRefresh() {
	a.message = ""
	a.refreshing = true
	a.renderStatus()
	a.refresh()
}

// refresh reloads the VM list in the background. A call made while a reload
// is running queues one more reload after it, so a change that lands
// mid-reload (e.g. a finished `limactl start`) isn't missed until the next
// tick. Safe to call from any goroutine.
func (a *app) refresh() {
	a.loadMu.Lock()
	defer a.loadMu.Unlock()
	if a.loading {
		a.reloadAgain = true
		return
	}
	a.loading = true
	go a.loadLoop()
}

func (a *app) loadLoop() {
	for {
		a.load()
		a.loadMu.Lock()
		again := a.reloadAgain
		a.reloadAgain, a.loading = false, again
		a.loadMu.Unlock()
		if !again {
			return
		}
	}
}

// load runs `limactl list` and shows the result.
func (a *app) load() {
	vms, err := loadVMs()
	a.tv.QueueUpdateDraw(func() {
		manual := a.refreshing
		a.refreshing = false
		if err != nil {
			a.setError(err)
			return
		}
		if manual {
			// A list takes milliseconds, so "Refreshing…" alone would just flicker.
			a.message = "[green]Refreshed[-]"
		}
		a.vms = vms
		a.lastUpdate = time.Now()
		a.render()
	})
}

// selectedVM returns a copy of the highlighted VM, or false if none is selected.
func (a *app) selectedVM() (instance, bool) {
	row, _ := a.table.GetSelection()
	if row < 1 || row > len(a.vms) {
		return instance{}, false
	}
	return a.vms[row-1], true
}

// render redraws everything that shows app state.
func (a *app) render() {
	a.renderTable()
	a.renderHeader()
	a.renderStatus()
}

func (a *app) setMessage(msg string) {
	a.message = msg
	a.renderStatus()
}

func (a *app) setError(err error) {
	a.setMessage("[red]" + tview.Escape(err.Error()) + "[-]")
}

func (a *app) renderStatus() {
	var parts []string
	if a.refreshing {
		parts = append(parts, "Refreshing…")
	}
	if a.message != "" {
		parts = append(parts, a.message)
	}
	if !a.lastUpdate.IsZero() {
		parts = append(parts, fmt.Sprintf("%d VMs, updated %s", len(a.vms), a.lastUpdate.Format("15:04:05")))
	}
	if !a.showHeader {
		parts = append(parts, statusHints())
	}
	a.statusBar.SetText(" " + strings.Join(parts, "  │  "))
}
