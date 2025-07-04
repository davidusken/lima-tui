package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// applyTheme applies the current theme to all UI components
func (a *App) applyTheme() {
	if a.lightMode {
		// Light mode colors (for white terminal backgrounds)
		tview.Styles.PrimitiveBackgroundColor = tcell.ColorWhite
		tview.Styles.ContrastBackgroundColor = tcell.ColorLightGray
		tview.Styles.MoreContrastBackgroundColor = tcell.ColorDarkGray
		tview.Styles.BorderColor = tcell.ColorDarkGray
		tview.Styles.TitleColor = tcell.ColorDarkBlue
		tview.Styles.GraphicsColor = tcell.ColorDarkGray
		tview.Styles.PrimaryTextColor = tcell.ColorBlack
		tview.Styles.SecondaryTextColor = tcell.ColorDarkGray
		tview.Styles.TertiaryTextColor = tcell.ColorGray
		tview.Styles.InverseTextColor = tcell.ColorWhite
	} else {
		// Dark mode colors (for dark terminal backgrounds)
		tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
		tview.Styles.ContrastBackgroundColor = tcell.ColorDarkBlue
		tview.Styles.MoreContrastBackgroundColor = tcell.ColorBlue
		tview.Styles.BorderColor = tcell.ColorWhite
		tview.Styles.TitleColor = tcell.ColorYellow
		tview.Styles.GraphicsColor = tcell.ColorWhite
		tview.Styles.PrimaryTextColor = tcell.ColorWhite
		tview.Styles.SecondaryTextColor = tcell.ColorLightGray
		tview.Styles.TertiaryTextColor = tcell.ColorGray
		tview.Styles.InverseTextColor = tcell.ColorBlack
	}
	
	a.applyTableTheme()
	a.applyComponentTheme()
}

// applyTableTheme applies theme to the table
func (a *App) applyTableTheme() {
	if a.lightMode {
		// Light mode table styling
		a.table.SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorWhite).
			Background(tcell.ColorBlue).
			Attributes(tcell.AttrBold))
		a.table.SetBorderColor(tcell.ColorDarkGray)
		a.table.SetTitleColor(tcell.ColorDarkBlue)
		a.table.SetBackgroundColor(tcell.ColorWhite)
	} else {
		// Dark mode table styling
		a.table.SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorAqua).
			Attributes(tcell.AttrBold))
		a.table.SetBorderColor(tcell.ColorWhite)
		a.table.SetTitleColor(tcell.ColorAqua)
		a.table.SetBackgroundColor(tcell.ColorBlack)
	}
}

// applyComponentTheme applies theme to other UI components
func (a *App) applyComponentTheme() {
	if a.lightMode {
		// Light mode component styling - better for iTerm2
		a.statusBar.SetBackgroundColor(tcell.ColorWhite)
		a.statusBar.SetTextColor(tcell.ColorDarkGray)
		a.helpText.SetBackgroundColor(tcell.ColorWhite)
		a.helpText.SetTextColor(tcell.ColorBlack)
		a.helpText.SetBorderColor(tcell.ColorDarkGray)
		a.helpText.SetTitleColor(tcell.ColorDarkBlue)
	} else {
		// Dark mode component styling - better for iTerm2
		a.statusBar.SetBackgroundColor(tcell.ColorBlack)
		a.statusBar.SetTextColor(tcell.ColorLightGray)
		a.helpText.SetBackgroundColor(tcell.ColorBlack)
		a.helpText.SetTextColor(tcell.ColorLightGray)
		a.helpText.SetBorderColor(tcell.ColorLightGray)
		a.helpText.SetTitleColor(tcell.ColorAqua)
	}
}

// ToggleTheme toggles between light and dark theme
func (a *App) ToggleTheme() {
	a.lightMode = !a.lightMode
	a.applyTheme()
	a.updateTable() // Refresh table to apply new colors
}

// Theme color helpers
func (a *App) getTextColor() tcell.Color {
	if a.lightMode {
		return tcell.ColorBlack
	}
	return tcell.ColorWhite
}

func (a *App) getHeaderColor() tcell.Color {
	if a.lightMode {
		return tcell.ColorDarkBlue
	}
	return tcell.ColorAqua
}

func (a *App) getStatusColor(status string) tcell.Color {
	switch status {
	case "Running":
		return tcell.ColorGreen
	case "Stopped":
		return tcell.ColorRed
	case "Starting":
		return tcell.ColorYellow
	case "Stopping":
		return tcell.ColorOrange
	default:
		return tcell.ColorGray
	}
}
