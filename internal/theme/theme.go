// Package theme has lima-tui's colour themes: the built-in dark and light
// themes, YAML skins layered on top of Dark, and config.yaml, which picks one.
package theme

import "github.com/gdamore/tcell/v2"

// Theme is a full set of UI colours.
type Theme struct {
	Name        string
	BG, FG, Dim tcell.Color
	HeaderFG    tcell.Color
	HeaderBG    tcell.Color
	Title       tcell.Color
	SelectedFG  tcell.Color
	SelectedBG  tcell.Color
	ButtonFG    tcell.Color
	ButtonBG    tcell.Color
	Running     tcell.Color
	Stopped     tcell.Color
	Starting    tcell.Color // also used while restarting
	Stopping    tcell.Color // also used while deleting
}

var (
	// Dark is the default theme, and the base every skin starts from.
	Dark = Theme{
		Name: "dark",
		BG:   tcell.ColorBlack, FG: tcell.ColorWhite, Dim: tcell.ColorLightGray,
		HeaderFG: tcell.ColorWhite, HeaderBG: tcell.ColorDarkGreen,
		Title:      tcell.ColorAqua,
		SelectedFG: tcell.ColorBlack, SelectedBG: tcell.ColorAqua,
		ButtonFG: tcell.ColorWhite, ButtonBG: tcell.ColorDarkBlue,
		Running: tcell.ColorGreen, Stopped: tcell.ColorRed,
		Starting: tcell.ColorYellow, Stopping: tcell.ColorOrange,
	}
	// Light is for terminals with a light background.
	Light = Theme{
		Name: "light",
		// W3C "darkgray" (#a9a9a9) is lighter than "gray"; dimgray reads on white.
		BG: tcell.ColorWhite, FG: tcell.ColorBlack, Dim: tcell.ColorDimGray,
		HeaderFG: tcell.ColorWhite, HeaderBG: tcell.ColorDarkSlateGray,
		Title:      tcell.ColorDarkBlue,
		SelectedFG: tcell.ColorWhite, SelectedBG: tcell.ColorBlue,
		ButtonFG: tcell.ColorBlack, ButtonBG: tcell.ColorLightGray,
		Running: tcell.ColorGreen, Stopped: tcell.ColorRed,
		// Yellow and orange are near invisible on white.
		Starting: tcell.ColorDarkGoldenrod, Stopping: tcell.ColorChocolate,
	}
)

// Builtins returns the themes that need no skin file, in the order t cycles
// them. Each call returns a new slice, so callers can't change each other's.
func Builtins() []Theme {
	return []Theme{Dark, Light}
}
