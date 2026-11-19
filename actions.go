package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/rivo/tview"
)

// Actions lima-tui runs on a VM. While one runs, the VM's status cell shows
// it instead of what limactl reports.
const (
	actStarting   = "Starting"
	actStopping   = "Stopping"
	actRestarting = "Restarting"
	actDeleting   = "Deleting"
)

// connect opens `limactl shell` for the selected VM, then returns to the UI.
func (a *App) connect() {
	vm, ok := a.selectedVM()
	if !ok {
		return
	}
	if vm.Status != statusRunning {
		a.setMessage(fmt.Sprintf("%s is %s; press s to start it first", vm.Name, vm.Status))
		return
	}

	a.app.Suspend(func() {
		cmd := exec.Command("limactl", "shell", vm.Name)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		// A non-zero exit is just the status of the user's last command in
		// the shell; only pause when the shell couldn't start at all.
		var exitErr *exec.ExitError
		if err := cmd.Run(); err != nil && !errors.As(err, &exitErr) {
			fmt.Printf("\nlimactl shell %s: %v\nPress Enter to return to lima-tui…", vm.Name, err)
			_, _ = fmt.Scanln()
		}
	})
	a.refresh()
}

// toggleRunning starts a stopped VM or stops a running one.
func (a *App) toggleRunning() {
	vm, ok := a.selectedIdleVM()
	if !ok {
		return
	}
	switch vm.Status {
	case statusRunning:
		a.runLimactl(vm.Name, actStopping, "Stopped", []string{"stop", vm.Name})
	case statusStopped:
		a.runLimactl(vm.Name, actStarting, "Started", []string{"start", vm.Name})
	default:
		a.setMessage(fmt.Sprintf("%s is %s; only running or stopped VMs can be started/stopped", vm.Name, vm.Status))
	}
}

func (a *App) restart() {
	vm, ok := a.selectedIdleVM()
	if !ok {
		return
	}
	if vm.Status != statusRunning {
		a.setMessage(fmt.Sprintf("%s is %s; only running VMs can be restarted", vm.Name, vm.Status))
		return
	}
	a.runLimactl(vm.Name, actRestarting, "Restarted",
		[]string{"stop", vm.Name}, []string{"start", vm.Name})
}

// confirmDelete asks before deleting the selected VM. It always uses
// `limactl delete --force`, so a VM that is running (or was started while
// the dialog was open) is killed instead of making the delete fail.
func (a *App) confirmDelete() {
	vm, ok := a.selectedIdleVM()
	if !ok {
		return
	}

	text := fmt.Sprintf("Delete VM %q?\n\nIts disk and configuration will be removed. This cannot be undone.", vm.Name)
	if vm.Status == statusRunning {
		text += "\n\nThe VM is running and will be shut down without a clean stop."
	}

	t := a.theme()
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Cancel", "Delete"}).
		SetBackgroundColor(t.bg).
		SetTextColor(t.fg).
		SetButtonBackgroundColor(t.buttonBg).
		SetButtonTextColor(t.buttonFg).
		SetDoneFunc(func(_ int, label string) {
			a.pages.RemovePage(pageConfirm)
			a.app.SetFocus(a.table)
			if label != "Delete" {
				return
			}
			// Another action may have started while the dialog was open.
			if a.busy[vm.Name] != "" {
				a.setMessage(fmt.Sprintf("%s is busy; not deleted", vm.Name))
				return
			}
			a.runLimactl(vm.Name, actDeleting, "Deleted",
				[]string{"delete", "--force", vm.Name})
		})

	a.pages.AddPage(pageConfirm, modal, false, true)
}

// selectedIdleVM is selectedVM, but refuses (with a message) a VM that already
// has a limactl command running from this UI. The VM list can lag behind by a
// refresh, so its status alone doesn't stop a second start/stop/delete.
func (a *App) selectedIdleVM() (VM, bool) {
	vm, ok := a.selectedVM()
	if !ok {
		return VM{}, false
	}
	if act := a.busy[vm.Name]; act != "" {
		a.setMessage(fmt.Sprintf("%s is busy (%s); wait for it to finish", vm.Name, act))
		return VM{}, false
	}
	return vm, true
}

// runLimactl runs the given limactl commands for VM name one after another in
// the background. act (e.g. actStarting) is shown in the VM's status cell and
// the status bar until they finish, then done ("Started") or the error.
// Must be called from the event loop.
func (a *App) runLimactl(name, act, done string, cmds ...[]string) {
	a.busy[name] = act
	a.setMessage("[yellow]" + tview.Escape(act+" "+name) + "…[-]")
	a.renderTable()
	go func() {
		var err error
		for _, args := range cmds {
			if err = limactl(args...); err != nil {
				break
			}
		}
		a.app.QueueUpdateDraw(func() {
			delete(a.busy, name)
			a.renderTable()
			if err != nil {
				a.setError(err)
			} else {
				a.setMessage("[green]" + tview.Escape(done+" "+name) + "[-]")
			}
		})
		a.refresh()
	}()
}
