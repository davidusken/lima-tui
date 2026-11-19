package app

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rivo/tview"

	"github.com/davidusken/lima-tui/internal/lima"
	"github.com/davidusken/lima-tui/internal/theme"
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

	vms         []lima.Instance
	lastUpdate  time.Time
	message     string
	refreshing  bool // a reload the user asked for is running
	showHelp    bool
	showHeader  bool
	version     string // lima-tui's
	limaVersion string
	home        string
	themes      []theme.Theme
	themeIdx    int
	busy        map[string]string // VM name -> action in progress (actStarting, …)

	// How the app calls limactl (lima.List and lima.Run); tests swap in fakes.
	// Set once in newApp, then only read.
	listVMs    func() ([]lima.Instance, error)
	runCommand func(args ...string) error

	loadMu      sync.Mutex
	loading     bool // a reload is running
	reloadAgain bool // reload once more when it finishes

	// manualSeq counts r presses, so a load can tell whether it started
	// after the latest one.
	manualSeq atomic.Uint64

	// suspended is set while a VM shell has the terminal. The event loop is
	// blocked then, so background refreshes would only pile up.
	suspended atomic.Bool
}

// Options is what lima-tui starts with.
type Options struct {
	Themes   []theme.Theme // must not be empty
	Theme    int           // index in Themes of the theme to start with
	Warnings []string      // shown in the status bar, e.g. skins that didn't load
	Version  string        // lima-tui's own version, for the header
}

// Run shows the UI until the user quits.
func Run(opts Options) error {
	return newApp(opts).run()
}

// newApp builds the UI.
func newApp(opts Options) *app {
	// Empty if unknown; shortenHome then leaves paths alone.
	home, _ := os.UserHomeDir()
	var notice string
	if len(opts.Warnings) > 0 {
		notice = colored(opts.Themes[opts.Theme].Stopped, strings.Join(opts.Warnings, "; "))
	}
	a := &app{
		version:    opts.Version,
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
		home:       home,
		themes:     opts.Themes,
		themeIdx:   opts.Theme,
		busy:       map[string]string{},
		message:    notice,
		listVMs:    lima.List,
		runCommand: lima.Run,
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
		v := lima.Version()
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

// quit stops the UI, but asks first while limactl commands started from it
// are still running: they carry on after lima-tui exits, with nobody to see
// whether they worked.
func (a *app) quit() {
	if len(a.busy) == 0 {
		a.tv.Stop()
		return
	}
	names := make([]string, 0, len(a.busy))
	for name := range a.busy {
		names = append(names, name)
	}
	sort.Strings(names)
	var running []string
	for _, name := range names {
		running = append(running, a.busy[name]+" "+name)
	}
	what, it := "A VM action is", "It keeps"
	if len(running) > 1 {
		what, it = "VM actions are", "They keep"
	}
	a.confirm(fmt.Sprintf("%s still running:\n\n%s\n\n%s running in the background if you quit, "+
		"but you won't see whether it worked. Quit anyway?", what, strings.Join(running, "\n"), it),
		"Quit", a.tv.Stop)
}

// manualRefresh is the r key: reload now and say so until it's done.
func (a *app) manualRefresh() {
	a.message = ""
	a.refreshing = true
	a.manualSeq.Add(1)
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
	seq := a.manualSeq.Load()
	vms, err := a.listVMs()
	a.tv.QueueUpdateDraw(func() {
		// Only a load that started after the last r press answers it. One
		// already running then shows older data; refresh queued another.
		manual := a.refreshing && seq == a.manualSeq.Load()
		if manual {
			a.refreshing = false
		}
		if err != nil {
			a.setError(err)
			return
		}
		if manual {
			// A list takes milliseconds, so "Refreshing…" alone would just flicker.
			a.message = colored(a.theme().Running, "Refreshed")
		}
		a.vms = vms
		a.lastUpdate = time.Now()
		a.render()
	})
}

// selectedVM returns a copy of the highlighted VM, or false if none is selected.
func (a *app) selectedVM() (lima.Instance, bool) {
	row, _ := a.table.GetSelection()
	if row < 1 || row > len(a.vms) {
		return lima.Instance{}, false
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
	a.setMessage(colored(a.theme().Stopped, err.Error()))
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
