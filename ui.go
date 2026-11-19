package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var columns = []string{"NAME", "STATUS", "SSH", "TYPE", "ARCH", "CPUS", "MEMORY", "DISK", "DIR"}

func (a *app) setupUI() {
	a.table.
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetBorder(true).
		SetTitle(" Lima VMs ")

	a.statusBar.SetDynamicColors(true)

	a.helpText.
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle(" Keys ")

	a.headerInfo.SetSelectable(false, false)
	a.headerKeys.SetSelectable(false, false)
	a.header.
		AddItem(a.headerInfo, headerInfoWidth, 0, false).
		AddItem(a.headerKeys, 0, 1, false)

	a.applyTheme()
	a.layout()
	a.render()
}

// layout (re)builds the main page; called when the header or help panel is toggled.
func (a *app) layout() {
	body := tview.NewFlex().AddItem(a.table, 0, 1, true)
	if a.showHelp {
		body.AddItem(a.helpText, 40, 0, false)
	}

	a.root.Clear()
	a.root.SetDirection(tview.FlexRow)
	if a.showHeader {
		a.root.AddItem(a.header, headerRows, 0, false)
	}
	a.root.
		AddItem(body, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)

	if !a.pages.HasPage(pageMain) {
		a.pages.AddPage(pageMain, a.root, true, true)
	}
}

func (a *app) toggleHelp() {
	a.showHelp = !a.showHelp
	a.layout()
	a.tv.SetFocus(a.table)
}

// closeHelp is Esc. It never quits, so a stray Esc is harmless.
func (a *app) closeHelp() {
	if a.showHelp {
		a.toggleHelp()
	}
}

// renderTable redraws the table from a.vms, keeping the same VM selected.
// The selected name is read from the table itself, which still shows the
// previous list: a.vms may already have been replaced, and looking the row up
// there would select whichever VM now sits at that position.
func (a *app) renderTable() {
	selected := ""
	if row, _ := a.table.GetSelection(); row >= 1 {
		if cell := a.table.GetCell(row, 0); cell != nil {
			selected = cell.Text
		}
	}
	t := a.theme()

	a.table.Clear()
	for col, name := range columns {
		a.table.SetCell(0, col, tview.NewTableCell(name).
			SetTextColor(t.headerFg).
			SetBackgroundColor(t.headerBg).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetExpansion(1))
	}

	if len(a.vms) == 0 {
		if !a.lastUpdate.IsZero() {
			a.table.SetCell(1, 0, tview.NewTableCell("No VMs found. Create one with: limactl start").
				SetTextColor(t.dim).SetSelectable(false))
		}
		return
	}

	selectRow := 1
	for i, vm := range a.vms {
		row := i + 1
		if vm.Name == selected {
			selectRow = row
		}
		status := vm.Status
		if act := a.busy[vm.Name]; act != "" {
			status = act
		}
		cells := []string{
			vm.Name,
			status,
			vm.sshEndpoint(),
			vm.VMType,
			vm.Arch,
			fmt.Sprint(vm.CPUs),
			humanBytes(vm.Memory),
			humanBytes(vm.Disk),
			// Cells parse [colour] tags and paths may contain brackets.
			// (Lima instance names can't, so the name stays raw.)
			tview.Escape(shortenHome(vm.Dir, a.home)),
		}
		for col, text := range cells {
			color := t.fg
			if col == 1 {
				color = t.statusColor(status)
				if status != vm.Status {
					text += "…"
				}
			}
			a.table.SetCell(row, col, tview.NewTableCell(text).SetTextColor(color).SetExpansion(1))
		}
	}
	a.table.Select(selectRow, 0)
}

func helpText(t theme) string {
	var b strings.Builder
	for _, k := range bindings {
		if k.section != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "[::b]%s[::-]\n", k.section)
		}
		fmt.Fprintf(&b, "  %-10s %s\n", tview.Escape(k.helpLabel()), k.desc)
	}
	fmt.Fprintf(&b, "\n[::b]Status[::-]\n  [%s]Running[-]  [%s]Stopped[-]\n  [%s]Starting[-] [%s]Stopping[-]\n",
		colorTag(t.running), colorTag(t.stopped), colorTag(t.starting), colorTag(t.stopping))
	fmt.Fprintf(&b, "\nRefreshes every %s.", refreshInterval)
	return b.String()
}
