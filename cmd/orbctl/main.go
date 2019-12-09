package main

import (
	"fmt"
	"os"
)

var (
	// Build arguments
	gitCommit = "none"
	gitTag    = "none"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.Stderr.Write([]byte(fmt.Sprintf("\x1b[0;31m%v\x1b[0m\n", r)))
			os.Exit(1)
		}
	}()

	rootCmd, rootValues := rootCommand()
	rootCmd.Version = fmt.Sprintf("%s %s\n", gitTag, gitCommit)
	rootCmd.AddCommand(
		takeoffCommand(rootValues),
		readSecretCommand(rootValues),
		writeSecretCommand(rootValues),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
