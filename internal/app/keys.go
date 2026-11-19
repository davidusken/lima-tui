package app

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// binding is one entry in the key map. The key handler, the help panel, the
// header hints and the status bar hint are all built from bindings, so they
// can't drift apart. README.md has its own copy of the table;
// TestReadmeListsEveryKey checks it.
type binding struct {
	section string // starts a new group in the help panel
	desc    string // help panel text
	hint    string // header hint text; empty keeps the binding out of the header
	status  bool   // also hinted in the status bar while the header is hidden
	keys    []key  // the first one is shown in the header
	action  func(*app)
	label   string // help panel key text for keys the table handles itself
}

// key is a special key (Enter, Ctrl+E) or, with code tcell.KeyRune, a character.
type key struct {
	code tcell.Key
	r    rune
}

func special(code tcell.Key) key { return key{code: code} }
func char(r rune) key            { return key{code: tcell.KeyRune, r: r} }

// bindings is filled in init: the actions refer back to it (via helpText),
// which Go doesn't allow in a package-level initializer.
var bindings []binding

func init() {
	bindings = []binding{
		{section: "Navigate", label: "↑/↓, j/k", desc: "Select VM"},
		{label: "g/G", desc: "First/last VM"},

		{section: "Actions", desc: "Open a shell in the VM", hint: "Shell",
			keys: []key{special(tcell.KeyEnter), char('c')}, action: (*app).connect},
		// Actions that change a VM need Ctrl, so a stray key press can't stop one.
		{desc: "Start/stop VM", hint: "Start/Stop", keys: []key{special(tcell.KeyCtrlS)}, action: (*app).toggleRunning},
		{desc: "Restart VM", hint: "Restart", keys: []key{special(tcell.KeyCtrlR)}, action: (*app).restart},
		{desc: "Delete VM (asks first)", hint: "Delete", keys: []key{special(tcell.KeyCtrlD)}, action: (*app).confirmDelete},
		{desc: "Refresh now", hint: "Refresh", keys: []key{char('r')}, action: (*app).manualRefresh},

		{section: "View", desc: "Next theme/skin", hint: "Theme", keys: []key{char('t')}, action: (*app).cycleTheme},
		{desc: "Toggle this help", hint: "Help", status: true,
			keys: []key{char('?'), char('h')}, action: (*app).toggleHelp},
		{desc: "Close this help", keys: []key{special(tcell.KeyEsc)}, action: (*app).closeHelp},
		{desc: "Toggle key header", hint: "Header", status: true,
			keys: []key{special(tcell.KeyCtrlE)}, action: (*app).toggleHeader},
		{desc: "Quit", hint: "Quit", status: true,
			keys: []key{char('q'), special(tcell.KeyCtrlC)}, action: (*app).quit},
	}
}

// name is the key as shown to the user: "q", "Enter", "Ctrl+E".
func (k key) name() string {
	if k.code == tcell.KeyRune {
		return string(k.r)
	}
	return strings.ReplaceAll(tcell.KeyNames[k.code], "Ctrl-", "Ctrl+")
}

func (k key) matches(ev *tcell.EventKey) bool {
	return ev.Key() == k.code && (k.code != tcell.KeyRune || ev.Rune() == k.r)
}

func (b binding) keyNames() []string {
	names := make([]string, len(b.keys))
	for i, k := range b.keys {
		names[i] = k.name()
	}
	return names
}

// helpLabel is the key column of the help panel.
func (b binding) helpLabel() string {
	if b.label != "" {
		return b.label
	}
	return strings.Join(b.keyNames(), ", ")
}

// hintKey is the key shown in the header, k9s style: the first one, lower
// case, "ctrl-e" rather than "Ctrl+E".
func (b binding) hintKey() string {
	return strings.ToLower(strings.ReplaceAll(b.keys[0].name(), "Ctrl+", "Ctrl-"))
}

// statusHints is the short key list the status bar shows when the header is
// hidden, e.g. "?: help  ctrl-e: header  q: quit".
func statusHints() string {
	var hints []string
	for _, b := range bindings {
		if b.status {
			hints = append(hints, b.hintKey()+": "+strings.ToLower(b.hint))
		}
	}
	return strings.Join(hints, "  ")
}

func (a *app) handleKey(event *tcell.EventKey) *tcell.EventKey {
	// Let the delete dialog handle its own keys.
	if front, _ := a.pages.GetFrontPage(); front != pageMain {
		return event
	}
	for _, b := range bindings {
		for _, k := range b.keys {
			if k.matches(event) {
				b.action(a)
				return nil
			}
		}
	}
	return event // navigation keys go to the table
}
