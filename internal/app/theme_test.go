package app

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestColorTag(t *testing.T) {
	if got := colorTag(tcell.ColorDefault); got != "-" {
		t.Errorf("default = %q", got)
	}
	if got := colorTag(tcell.ColorGreen); got != "green" {
		t.Errorf("green = %q", got)
	}
	if got := colorTag(tcell.NewHexColor(0x2aa198)); got != "#2AA198" {
		t.Errorf("hex = %q", got)
	}
}
