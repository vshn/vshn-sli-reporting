package main

import (
	"time"

	"github.com/vshn/vshn-sli-reporting/pkg/cmd"
)

var (
	// these variables are populated by Goreleaser when releasing
	version = "unknown"
	commit  = "-dirty-"
	date    = time.Now().Format("2006-01-02")

	appName     = "vshn-sli-reporting"
	appLongName = "VSHN SLI Reporting"
)

func main() {
	cmd.Execute()
}
