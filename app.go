package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// refreshInterval is how often the VM list is reloaded in the background.
const refreshInterval = 3 * time.Second

const (
	pageMain    = "main"
	pageConfirm = "confirm"
)

// App holds the UI and the last known VM list. Fields above loadMu are only
// touched from the tview event loop (directly or via QueueUpdateDraw).
type App struct {
	app        *tview.Application
	pages      *tview.Pages
	root       *tview.Flex
	table      *tview.Table
	statusBar  *tview.TextView
	helpText   *tview.TextView
	header     *tview.Flex
	headerInfo *tview.Table
	headerKeys *tview.Table

	vms         []VM
	lastUpdate  time.Time
	message     string
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
}

// NewApp builds the UI. themes must not be empty; idx is the theme to start
// with and notice, if set, is shown in the status bar.
func NewApp(themes []theme, idx int, notice string) *App {
	a := &App{
		app:        tview.NewApplication(),
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
	a.app.SetInputCapture(a.handleKey)
	return a
}

// Run shows the UI until the user quits.
func (a *App) Run() error {
	stop := make(chan struct{})
	defer close(stop)

	a.refresh()
	go func() {
		v := LimaVersion()
		a.app.QueueUpdateDraw(func() {
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
				a.refresh()
			case <-stop:
				return
			}
		}
	}()

	return a.app.SetRoot(a.pages, true).Run()
}

func (a *App) handleKey(event *tcell.EventKey) *tcell.EventKey {
	// Let the delete dialog handle its own keys.
	if front, _ := a.pages.GetFrontPage(); front != pageMain {
		return event
	}

	switch event.Key() {
	case tcell.KeyCtrlC:
		a.app.Stop()
		return nil
	case tcell.KeyEsc:
		// Closes the help panel; never quits, so a stray Esc is harmless.
		if a.showHelp {
			a.toggleHelp()
		}
		return nil
	case tcell.KeyCtrlE:
		a.toggleHeader()
		return nil
	case tcell.KeyEnter:
		a.connect()
		return nil
	}

	switch event.Rune() {
	case 'q':
		a.app.Stop()
	case 'h', '?':
		a.toggleHelp()
	case 'c':
		a.connect()
	case 's':
		a.toggleRunning()
	case 'R':
		a.restart()
	case 'd':
		a.confirmDelete()
	case 'r':
		a.setMessage("Refreshing…")
		a.refresh()
	case 't':
		a.cycleTheme()
	default:
		return event // navigation keys go to the table
	}
	return nil
}

// refresh reloads the VM list in the background. A call made while a reload
// is running queues one more reload after it, so a change that lands
// mid-reload (e.g. a finished `limactl start`) isn't missed until the next
// tick. Safe to call from any goroutine.
func (a *App) refresh() {
	a.loadMu.Lock()
	defer a.loadMu.Unlock()
	if a.loading {
		a.reloadAgain = true
		return
	}
	a.loading = true
	go a.loadLoop()
}

func (a *App) loadLoop() {
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
func (a *App) load() {
	vms, err := LoadVMs()
	a.app.QueueUpdateDraw(func() {
		if err != nil {
			a.setError(err)
			return
		}
		if a.message == "Refreshing…" {
			a.message = ""
		}
		a.vms = vms
		a.lastUpdate = time.Now()
		a.renderTable()
		a.renderHeader()
		a.renderStatus()
	})
}

// selectedVM returns a copy of the highlighted VM, or false if none is selected.
func (a *App) selectedVM() (VM, bool) {
	row, _ := a.table.GetSelection()
	if row < 1 || row > len(a.vms) {
		return VM{}, false
	}
	return a.vms[row-1], true
}

func (a *App) setMessage(msg string) {
	a.message = msg
	a.renderStatus()
}

func (a *App) setError(err error) {
	a.setMessage("[red]" + tview.Escape(err.Error()) + "[-]")
}

func (a *App) renderStatus() {
	var parts []string
	if a.message != "" {
		parts = append(parts, a.message)
	}
	if !a.lastUpdate.IsZero() {
		parts = append(parts, fmt.Sprintf("%d VMs, updated %s", len(a.vms), a.lastUpdate.Format("15:04:05")))
	}
	if !a.showHeader {
		parts = append(parts, "?: help  ctrl-e: header  q: quit")
	}
	a.statusBar.SetText(" " + strings.Join(parts, "  │  "))
}
