package app

import (
	"os"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/davidusken/lima-tui/internal/lima"
)

func TestBindingsDontOverlap(t *testing.T) {
	seen := map[string]string{}
	for _, b := range bindings {
		for _, k := range b.keyNames() {
			if other, ok := seen[k]; ok {
				t.Errorf("key %s is bound to both %q and %q", k, other, b.desc)
			}
			seen[k] = b.desc
		}
	}
}

func TestBindingsAreComplete(t *testing.T) {
	for _, b := range bindings {
		hasKeys := len(b.keys) > 0
		switch {
		case b.action != nil && !hasKeys:
			t.Errorf("%q has an action but no keys", b.desc)
		case b.action == nil && hasKeys:
			t.Errorf("%q has keys but no action", b.desc)
		case b.action == nil && b.label == "":
			t.Errorf("%q has neither an action nor a help label", b.desc)
		case b.hint != "" && b.action == nil:
			t.Errorf("%q is hinted in the header but does nothing", b.desc)
		}
	}
}

// README.md keeps its own key table; every key must be in it.
func TestReadmeListsEveryKey(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bindings {
		for _, k := range b.keyNames() {
			if !strings.Contains(string(readme), "`"+k+"`") {
				t.Errorf("README.md doesn't list key `%s` (%s)", k, b.desc)
			}
		}
	}
}

func TestHandleKeyRunsBinding(t *testing.T) {
	a := newTestApp()
	a.handleKey(tcell.NewEventKey(tcell.KeyRune, '?', tcell.ModNone))
	if !a.showHelp {
		t.Error("? didn't open the help panel")
	}
	a.handleKey(tcell.NewEventKey(tcell.KeyCtrlE, 0, tcell.ModCtrl))
	if a.showHeader {
		t.Error("Ctrl+E didn't hide the header")
	}
	if ev := tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone); a.handleKey(ev) != ev {
		t.Error("j should go through to the table")
	}
}

// VM-changing actions need Ctrl; the plain letter does nothing.
func TestVMActionsNeedCtrl(t *testing.T) {
	a := newTestApp()
	a.vms = []lima.Instance{{Name: "vm1", Status: lima.StatusRunning}}
	a.renderTable()

	a.handleKey(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone))
	if a.pages.HasPage(pageConfirm) {
		t.Error("plain d opened the delete dialog")
	}
	a.handleKey(tcell.NewEventKey(tcell.KeyCtrlD, 0, tcell.ModCtrl))
	if !a.pages.HasPage(pageConfirm) {
		t.Error("Ctrl+D didn't open the delete dialog")
	}
}

func TestStatusHints(t *testing.T) {
	if got, want := statusHints(), "?: help  ctrl-e: header  q: quit"; got != want {
		t.Errorf("statusHints() = %q, want %q", got, want)
	}
}

func TestHeaderShowsHints(t *testing.T) {
	a := newTestApp()
	if got := a.headerKeys.GetCell(0, 0).Text; got != "<enter>" {
		t.Errorf("first header hint = %q, want <enter>", got)
	}
}
