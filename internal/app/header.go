package app

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/davidusken/lima-tui/internal/lima"
)

const (
	headerRows      = 4  // hint rows per column; also the header height
	headerInfoWidth = 28 // width of the info block on the left
	headerColumnGap = "        "
)

// renderHeader fills the info block and the key hint grid.
func (a *app) renderHeader() {
	t := a.theme()

	running := 0
	for _, vm := range a.vms {
		if vm.Status == lima.StatusRunning {
			running++
		}
	}
	limaVer := a.limaVersion
	if limaVer == "" {
		limaVer = "…"
	}
	info := []struct{ label, value string }{
		{"lima-tui", a.version},
		{"Lima", limaVer},
		{"VMs", fmt.Sprintf("%d running / %d", running, len(a.vms))},
		{"Theme", tview.Escape(t.Name)}, // skin names come from file names
	}

	a.headerInfo.Clear()
	for row, kv := range info {
		a.headerInfo.SetCell(row, 0, tview.NewTableCell(kv.label+":").SetTextColor(t.Title))
		a.headerInfo.SetCell(row, 1, tview.NewTableCell(kv.value).SetTextColor(t.FG))
	}

	a.headerKeys.Clear()
	i := 0
	for _, b := range bindings {
		if b.hint == "" {
			continue
		}
		row, col := i%headerRows, (i/headerRows)*2
		a.headerKeys.SetCell(row, col, tview.NewTableCell(tview.Escape("<"+b.hintKey()+">")).SetTextColor(t.Title))
		// Wide gap after each column so a description isn't read as belonging
		// to the next column's key.
		a.headerKeys.SetCell(row, col+1, tview.NewTableCell(b.hint+headerColumnGap).SetTextColor(t.Dim))
		i++
	}
}

func (a *app) toggleHeader() {
	a.showHeader = !a.showHeader
	a.layout()
	a.renderStatus()
	a.tv.SetFocus(a.table)
}
