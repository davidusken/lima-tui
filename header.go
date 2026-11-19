package main

import (
	"fmt"

	"github.com/rivo/tview"
)

// headerHints are the key hints shown in the header, k9s style. Keep them in
// sync with handleKey and keyHelp.
var headerHints = []struct{ key, desc string }{
	{"enter", "Shell"},
	{"s", "Start/Stop"},
	{"R", "Restart"},
	{"d", "Delete"},
	{"r", "Refresh"},
	{"t", "Theme"},
	{"?", "Help"},
	{"ctrl-e", "Header"},
	{"q", "Quit"},
}

const (
	headerRows      = 4  // hint rows per column; also the header height
	headerInfoWidth = 28 // width of the info block on the left
	headerColumnGap = "        "
)

// renderHeader fills the info block and the key hint grid.
func (a *App) renderHeader() {
	t := a.theme()

	running := 0
	for _, vm := range a.vms {
		if vm.Status == statusRunning {
			running++
		}
	}
	lima := a.limaVersion
	if lima == "" {
		lima = "…"
	}
	info := []struct{ label, value string }{
		{"lima-tui", version},
		{"Lima", lima},
		{"VMs", fmt.Sprintf("%d running / %d", running, len(a.vms))},
		{"Theme", tview.Escape(t.name)}, // skin names come from file names
	}

	a.headerInfo.Clear()
	for row, kv := range info {
		a.headerInfo.SetCell(row, 0, tview.NewTableCell(kv.label+":").SetTextColor(t.title))
		a.headerInfo.SetCell(row, 1, tview.NewTableCell(kv.value).SetTextColor(t.fg))
	}

	a.headerKeys.Clear()
	for i, h := range headerHints {
		row, col := i%headerRows, (i/headerRows)*2
		a.headerKeys.SetCell(row, col, tview.NewTableCell(tview.Escape("<"+h.key+">")).SetTextColor(t.title))
		// Wide gap after each column so a description isn't read as belonging
		// to the next column's key.
		a.headerKeys.SetCell(row, col+1, tview.NewTableCell(h.desc+headerColumnGap).SetTextColor(t.dim))
	}
}

func (a *App) toggleHeader() {
	a.showHeader = !a.showHeader
	a.layout()
	a.renderStatus()
	a.app.SetFocus(a.table)
}
