package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// connectToSelected connects to the selected VM
func (a *App) connectToSelected() {
	vm := a.GetSelectedVM()
	if vm == nil {
		a.UpdateStatus("[red]No VM selected")
		return
	}
	
	if vm.Status != "Running" {
		a.UpdateStatus(fmt.Sprintf("VM '%s' is not running (status: %s)", vm.Name, vm.Status))
		return
	}
	
	a.UpdateStatus(fmt.Sprintf("Connecting to VM '%s'...", vm.Name))
	a.app.Suspend(func() {
		cmd := exec.Command("limactl", "shell", vm.Name)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error connecting to VM: %v\n", err)
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
		}
	})
}

// toggleVMState toggles the state of the selected VM (start/stop)
func (a *App) toggleVMState() {
	vm := a.GetSelectedVM()
	if vm == nil {
		a.UpdateStatus("[red]No VM selected")
		return
	}
	
	var action string
	var cmd *exec.Cmd
	
	switch vm.Status {
	case "Running":
		action = "Stopping"
		cmd = exec.Command("limactl", "stop", vm.Name)
	case "Stopped":
		action = "Starting"
		cmd = exec.Command("limactl", "start", vm.Name)
	default:
		a.UpdateStatus(fmt.Sprintf("Cannot toggle VM '%s' in state '%s'", vm.Name, vm.Status))
		return
	}
	
	a.UpdateStatus(fmt.Sprintf("%s VM '%s'...", action, vm.Name))
	
	go func() {
		err := cmd.Run()
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.UpdateStatus(fmt.Sprintf("Failed to %s VM '%s': %v", strings.ToLower(action), vm.Name, err))
			} else {
				a.UpdateStatus(fmt.Sprintf("Successfully %s VM '%s'", strings.ToLower(action), vm.Name))
				// Refresh the list after a short delay
				time.Sleep(500 * time.Millisecond)
				a.LoadVMs()
			}
		})
	}()
}

// restartSelected restarts the selected VM
func (a *App) restartSelected() {
	vm := a.GetSelectedVM()
	if vm == nil {
		a.UpdateStatus("[red]No VM selected")
		return
	}
	
	if vm.Status != "Running" {
		a.UpdateStatus(fmt.Sprintf("VM '%s' is not running (status: %s)", vm.Name, vm.Status))
		return
	}
	
	a.UpdateStatus(fmt.Sprintf("Restarting VM '%s'...", vm.Name))
	
	go func() {
		// Stop first
		stopCmd := exec.Command("limactl", "stop", vm.Name)
		if err := stopCmd.Run(); err != nil {
			a.app.QueueUpdateDraw(func() {
				a.UpdateStatus(fmt.Sprintf("Failed to stop VM '%s': %v", vm.Name, err))
			})
			return
		}
		
		// Wait a bit
		time.Sleep(2 * time.Second)
		
		// Start again
		startCmd := exec.Command("limactl", "start", vm.Name)
		err := startCmd.Run()
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.UpdateStatus(fmt.Sprintf("Failed to restart VM '%s': %v", vm.Name, err))
			} else {
				a.UpdateStatus(fmt.Sprintf("Successfully restarted VM '%s'", vm.Name))
				// Refresh the list after a short delay
				time.Sleep(500 * time.Millisecond)
				a.LoadVMs()
			}
		})
	}()
}

// deleteSelected deletes the selected VM with confirmation
func (a *App) deleteSelected() {
	vm := a.GetSelectedVM()
	if vm == nil {
		a.UpdateStatus("[red]No VM selected")
		return
	}
	
	// Show confirmation dialog
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure you want to delete VM '%s'?\n\nThis action cannot be undone.", vm.Name)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			// Restore the original layout
			a.SetupLayout()
			if buttonLabel == "Delete" {
				a.performDelete(vm)
			}
		})
	
	// Apply theme to modal
	if a.lightMode {
		modal.SetBackgroundColor(tcell.ColorWhite)
		modal.SetTextColor(tcell.ColorBlack)
		modal.SetButtonBackgroundColor(tcell.ColorLightGray)
		modal.SetButtonTextColor(tcell.ColorBlack)
	} else {
		modal.SetBackgroundColor(tcell.ColorBlack)
		modal.SetTextColor(tcell.ColorWhite)
		modal.SetButtonBackgroundColor(tcell.ColorDarkBlue)
		modal.SetButtonTextColor(tcell.ColorWhite)
	}
	
	// Show the modal
	a.app.SetRoot(modal, true)
}

// performDelete actually deletes the VM
func (a *App) performDelete(vm *VM) {
	a.UpdateStatus(fmt.Sprintf("Deleting VM '%s'...", vm.Name))
	
	go func() {
		cmd := exec.Command("limactl", "delete", vm.Name)
		err := cmd.Run()
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.UpdateStatus(fmt.Sprintf("Failed to delete VM '%s': %v", vm.Name, err))
			} else {
				a.UpdateStatus(fmt.Sprintf("Successfully deleted VM '%s'", vm.Name))
				// Refresh the list after a short delay
				time.Sleep(500 * time.Millisecond)
				a.LoadVMs()
			}
		})
	}()
}
