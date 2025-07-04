package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
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
