package main

import (
	"fmt"
	"os"

	"github.com/yourusername/syseng-agent/cmd"
)

// Build information (set by build script)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("iteasy-ai-agent %s\n", version)
		fmt.Printf("Built: %s\n", buildTime)
		fmt.Printf("Commit: %s\n", gitCommit)
		os.Exit(0)
	}

	cmd.Execute()
}
