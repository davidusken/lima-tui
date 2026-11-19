package main

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func newTestApp() *App {
	return NewApp(builtinThemes, 0, "")
}

func vmsNamed(names ...string) []VM {
	vms := make([]VM, len(names))
	for i, n := range names {
		vms[i] = VM{Name: n, Status: statusStopped}
	}
	return vms
}

func TestRenderTableKeepsSelectedVM(t *testing.T) {
	a := newTestApp()
	a.vms = vmsNamed("b", "c")
	a.renderTable()
	a.table.Select(2, 0) // c

	// A VM sorting before the selection appears, as on a refresh.
	a.vms = vmsNamed("a", "b", "c")
	a.renderTable()

	vm, ok := a.selectedVM()
	if !ok || vm.Name != "c" {
		t.Fatalf("selected %q after refresh, want c", vm.Name)
	}
}

func TestRenderTableSelectionGone(t *testing.T) {
	a := newTestApp()
	a.vms = vmsNamed("a", "b", "c")
	a.renderTable()
	a.table.Select(3, 0) // c

	a.vms = vmsNamed("a", "b")
	a.renderTable()

	if vm, _ := a.selectedVM(); vm.Name != "a" {
		t.Fatalf("selected %q after the selected VM went away, want a", vm.Name)
	}
}

func TestBusyVMRefusesActions(t *testing.T) {
	for name, action := range map[string]func(*App){
		"toggleRunning": (*App).toggleRunning,
		"restart":       (*App).restart,
		"confirmDelete": (*App).confirmDelete,
	} {
		t.Run(name, func(t *testing.T) {
			a := newTestApp()
			a.vms = []VM{{Name: "vm1", Status: statusRunning}}
			a.renderTable()
			a.busy["vm1"] = actStarting

			action(a)

			if !strings.Contains(a.message, "busy") {
				t.Errorf("message = %q, want a busy notice", a.message)
			}
			if a.pages.HasPage(pageConfirm) {
				t.Error("delete dialog opened for a busy VM")
			}
		})
	}
}

func TestBusyVMShowsAction(t *testing.T) {
	a := newTestApp()
	a.vms = []VM{{Name: "vm1", Status: statusStopped}}
	a.busy["vm1"] = actStarting
	a.renderTable()

	if got := a.table.GetCell(1, 1).Text; got != "Starting…" {
		t.Errorf("status cell = %q, want Starting…", got)
	}

	delete(a.busy, "vm1")
	a.renderTable()
	if got := a.table.GetCell(1, 1).Text; got != statusStopped {
		t.Errorf("status cell = %q after the action, want %s", got, statusStopped)
	}
}

func TestEscClosesHelpOnly(t *testing.T) {
	a := newTestApp()
	a.toggleHelp()
	a.handleKey(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone))
	if a.showHelp {
		t.Error("Esc didn't close the help panel")
	}
}
