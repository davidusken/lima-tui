package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/davidusken/lima-tui/internal/lima"
	"github.com/davidusken/lima-tui/internal/theme"
)

func (a *app) theme() theme.Theme {
	return a.themes[a.themeIdx]
}

// statusColor is the text colour for a VM status cell: a limactl status, or
// the action lima-tui is running on the VM.
func statusColor(t theme.Theme, status string) tcell.Color {
	switch status {
	case lima.StatusRunning:
		return t.Running
	case lima.StatusStopped:
		return t.Stopped
	case actStarting, actRestarting:
		return t.Starting
	case actStopping, actDeleting:
		return t.Stopping
	default:
		return t.FG
	}
}

// colorTag returns c as a tview colour tag value ("green", "#c0caf5", or "-"
// for the terminal default).
func colorTag(c tcell.Color) string {
	if !c.Valid() {
		return "-"
	}
	return c.String()
}

// colored returns text, escaped, as a tview string in colour c.
func colored(c tcell.Color, text string) string {
	return "[" + colorTag(c) + "]" + tview.Escape(text) + "[-]"
}

// applyTheme colours every widget for the current theme.
func (a *app) applyTheme() {
	t := a.theme()

	// Defaults for widgets created later (e.g. the delete dialog).
	tview.Styles.PrimitiveBackgroundColor = t.BG
	tview.Styles.ContrastBackgroundColor = t.ButtonBG
	tview.Styles.PrimaryTextColor = t.FG
	tview.Styles.SecondaryTextColor = t.Dim
	tview.Styles.BorderColor = t.Dim
	tview.Styles.TitleColor = t.Title

	a.table.SetBackgroundColor(t.BG)
	a.table.SetBorderColor(t.Dim)
	a.table.SetTitleColor(t.Title)
	a.table.SetSelectedStyle(tcell.StyleDefault.Foreground(t.SelectedFG).Background(t.SelectedBG).Bold(true))

	a.statusBar.SetBackgroundColor(t.BG)
	a.statusBar.SetTextColor(t.Dim)

	a.helpText.SetBackgroundColor(t.BG)
	a.helpText.SetTextColor(t.FG)
	a.helpText.SetBorderColor(t.Dim)
	a.helpText.SetTitleColor(t.Title)
	a.helpText.SetText(helpText(t))

	a.root.SetBackgroundColor(t.BG)
	a.header.SetBackgroundColor(t.BG)
	a.headerInfo.SetBackgroundColor(t.BG)
	a.headerKeys.SetBackgroundColor(t.BG)
}

// cycleTheme switches to the next built-in theme or loaded skin.
func (a *app) cycleTheme() {
	a.themeIdx = (a.themeIdx + 1) % len(a.themes)
	a.applyTheme()
	a.message = "Theme: " + tview.Escape(a.theme().Name)
	a.render()
}
