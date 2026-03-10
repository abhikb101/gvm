package main

import (
	"os"

	"github.com/gvm-tools/gvm/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	code := cmd.Execute()
	os.Exit(code)
}
