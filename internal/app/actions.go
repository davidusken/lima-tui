package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/rivo/tview"

	"github.com/davidusken/lima-tui/internal/lima"
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
func (a *app) connect() {
	vm, ok := a.selectedVM()
	if !ok {
		return
	}
	if vm.Status != lima.StatusRunning {
		a.setMessage(fmt.Sprintf("%s is %s; press Ctrl+S to start it first", vm.Name, vm.Status))
		return
	}

	a.suspended.Store(true)
	defer a.suspended.Store(false)
	a.tv.Suspend(func() {
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
func (a *app) toggleRunning() {
	vm, ok := a.selectedIdleVM()
	if !ok {
		return
	}
	switch vm.Status {
	case lima.StatusRunning:
		a.runLimactl(vm.Name, actStopping, "Stopped", "stop", vm.Name)
	case lima.StatusStopped:
		a.runLimactl(vm.Name, actStarting, "Started", "start", vm.Name)
	default:
		a.setMessage(fmt.Sprintf("%s is %s; only running or stopped VMs can be started/stopped", vm.Name, vm.Status))
	}
}

func (a *app) restart() {
	vm, ok := a.selectedIdleVM()
	if !ok {
		return
	}
	if vm.Status != lima.StatusRunning {
		a.setMessage(fmt.Sprintf("%s is %s; only running VMs can be restarted", vm.Name, vm.Status))
		return
	}
	a.runLimactl(vm.Name, actRestarting, "Restarted", "restart", vm.Name)
}

// confirmDelete asks before deleting the selected VM. It always uses
// `limactl delete --force`, so a VM that is running (or was started while
// the dialog was open) is killed instead of making the delete fail.
func (a *app) confirmDelete() {
	vm, ok := a.selectedIdleVM()
	if !ok {
		return
	}

	text := fmt.Sprintf("Delete VM %q?\n\nIts disk and configuration will be removed. This cannot be undone.", vm.Name)
	if vm.Status == lima.StatusRunning {
		text += "\n\nThe VM is running and will be shut down without a clean stop."
	}

	a.confirm(text, "Delete", func() {
		// Another action may have started while the dialog was open.
		if a.busy[vm.Name] != "" {
			a.setMessage(fmt.Sprintf("%s is busy; not deleted", vm.Name))
			return
		}
		a.runLimactl(vm.Name, actDeleting, "Deleted", "delete", "--force", vm.Name)
	})
}

// confirm shows a dialog with text and the buttons Cancel and ok, and calls
// onOK if ok is chosen. Cancel has the focus, so Enter is the safe answer.
func (a *app) confirm(text, ok string, onOK func()) {
	t := a.theme()
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Cancel", ok}).
		SetBackgroundColor(t.BG).
		SetTextColor(t.FG).
		SetButtonBackgroundColor(t.ButtonBG).
		SetButtonTextColor(t.ButtonFG).
		SetDoneFunc(func(_ int, label string) {
			a.pages.RemovePage(pageConfirm)
			a.tv.SetFocus(a.table)
			if label == ok {
				onOK()
			}
		})
	a.pages.AddPage(pageConfirm, modal, false, true)
}

// selectedIdleVM is selectedVM, but refuses (with a message) a VM that already
// has a limactl command running from this UI. The VM list can lag behind by a
// refresh, so its status alone doesn't stop a second start/stop/delete.
func (a *app) selectedIdleVM() (lima.Instance, bool) {
	vm, ok := a.selectedVM()
	if !ok {
		return lima.Instance{}, false
	}
	if act := a.busy[vm.Name]; act != "" {
		a.setMessage(fmt.Sprintf("%s is busy (%s); wait for it to finish", vm.Name, act))
		return lima.Instance{}, false
	}
	return vm, true
}

// runLimactl runs `limactl <args>` for VM name in the background. act (e.g.
// actStarting) is shown in the VM's status cell and the status bar until it
// finishes, then done ("Started") or the error. Must be called from the
// event loop.
func (a *app) runLimactl(name, act, done string, args ...string) {
	a.busy[name] = act
	a.setMessage(colored(statusColor(a.theme(), act), act+" "+name+"…"))
	a.renderTable()
	go func() {
		err := a.runCommand(args...)
		a.tv.QueueUpdateDraw(func() {
			delete(a.busy, name)
			a.renderTable()
			if err != nil {
				a.setError(err)
			} else {
				a.setMessage(colored(a.theme().Running, done+" "+name))
			}
		})
		a.refresh()
	}()
}
