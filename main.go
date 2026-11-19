package main

import (
	"fmt"
	"log"
	"os"
)

// Set at build time by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("lima-tui %s (commit %s, built %s)\n", version, commit, date)
			return
		case "-h", "--help", "help":
			fmt.Println("Usage: lima-tui [--version]\n\nA terminal UI for Lima VMs. Press h or ? inside the app for key bindings.")
			return
		}
	}

	for {
		app := NewApp()
		exitCode := app.Run()
		
		// If app.Run() returns an error, exit
		if exitCode != nil {
			log.Fatal(exitCode)
		}
		
		// Check if we should exit or restart
		if app.ShouldExit() {
			fmt.Println("Exiting application...")
			os.Exit(0)
		}
		
		// If we get here, user wanted to connect to a VM and we've returned
		// The application loop will restart to show the VM list again
		fmt.Println("Returned from VM connection. Restarting application...")
	}
}
