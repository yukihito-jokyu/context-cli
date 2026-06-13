package main

import (
	"fmt"
	"os"

	"github.com/yukihito-jokyu/context-cli/pkg/cmd"
)

func main() {
	f := cmd.NewFactory()
	rootCmd := cmd.NewCmdRoot(f)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
