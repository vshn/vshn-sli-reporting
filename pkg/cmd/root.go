package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	appName     = "vshn-sli-reporting"
	appLongName = "VSHN SLI Reporting"
)

var rootCmd = &cobra.Command{
	Use:   appName,
	Short: appLongName,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
