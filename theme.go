package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// theme is a full set of UI colours. Built-in themes are below; user skins
// (skin.go) are layered on top of darkTheme.
type theme struct {
	name        string
	bg, fg, dim tcell.Color
	headerFg    tcell.Color
	headerBg    tcell.Color
	title       tcell.Color
	selectedFg  tcell.Color
	selectedBg  tcell.Color
	buttonFg    tcell.Color
	buttonBg    tcell.Color
	running     tcell.Color
	stopped     tcell.Color
	starting    tcell.Color
	stopping    tcell.Color
}

var (
	darkTheme = theme{
		name: "dark",
		bg:   tcell.ColorBlack, fg: tcell.ColorWhite, dim: tcell.ColorLightGray,
		headerFg: tcell.ColorWhite, headerBg: tcell.ColorDarkGreen,
		title:      tcell.ColorAqua,
		selectedFg: tcell.ColorBlack, selectedBg: tcell.ColorAqua,
		buttonFg: tcell.ColorWhite, buttonBg: tcell.ColorDarkBlue,
		running: tcell.ColorGreen, stopped: tcell.ColorRed,
		starting: tcell.ColorYellow, stopping: tcell.ColorOrange,
	}
	lightTheme = theme{
		name: "light",
		bg:   tcell.ColorWhite, fg: tcell.ColorBlack, dim: tcell.ColorDarkGray,
		headerFg: tcell.ColorWhite, headerBg: tcell.ColorDarkSlateGray,
		title:      tcell.ColorDarkBlue,
		selectedFg: tcell.ColorWhite, selectedBg: tcell.ColorBlue,
		buttonFg: tcell.ColorBlack, buttonBg: tcell.ColorLightGray,
		running: tcell.ColorGreen, stopped: tcell.ColorRed,
		starting: tcell.ColorYellow, stopping: tcell.ColorOrange,
	}
	builtinThemes = []theme{darkTheme, lightTheme}
)

func (a *app) theme() theme {
	return a.themes[a.themeIdx]
}

// statusColor is the text colour for a VM status cell: a limactl status, or
// the action lima-tui is running on the VM.
func (t theme) statusColor(status string) tcell.Color {
	switch status {
	case statusRunning:
		return t.running
	case statusStopped:
		return t.stopped
	case actStarting, actRestarting:
		return t.starting
	case actStopping, actDeleting:
		return t.stopping
	default:
		return t.fg
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

// applyTheme colours every widget for the current theme.
func (a *app) applyTheme() {
	t := a.theme()

	// Defaults for widgets created later (e.g. the delete dialog).
	tview.Styles.PrimitiveBackgroundColor = t.bg
	tview.Styles.ContrastBackgroundColor = t.buttonBg
	tview.Styles.PrimaryTextColor = t.fg
	tview.Styles.SecondaryTextColor = t.dim
	tview.Styles.BorderColor = t.dim
	tview.Styles.TitleColor = t.title

	a.table.SetBackgroundColor(t.bg)
	a.table.SetBorderColor(t.dim)
	a.table.SetTitleColor(t.title)
	a.table.SetSelectedStyle(tcell.StyleDefault.Foreground(t.selectedFg).Background(t.selectedBg).Bold(true))

	a.statusBar.SetBackgroundColor(t.bg)
	a.statusBar.SetTextColor(t.dim)

	a.helpText.SetBackgroundColor(t.bg)
	a.helpText.SetTextColor(t.fg)
	a.helpText.SetBorderColor(t.dim)
	a.helpText.SetTitleColor(t.title)
	a.helpText.SetText(helpText(t))

	a.root.SetBackgroundColor(t.bg)
	a.header.SetBackgroundColor(t.bg)
	a.headerInfo.SetBackgroundColor(t.bg)
	a.headerKeys.SetBackgroundColor(t.bg)
}

// cycleTheme switches to the next built-in theme or loaded skin.
func (a *app) cycleTheme() {
	a.themeIdx = (a.themeIdx + 1) % len(a.themes)
	a.applyTheme()
	a.message = "Theme: " + tview.Escape(a.theme().name)
	a.render()
}
