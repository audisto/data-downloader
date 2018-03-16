//go:generate statik -src=./web/static -dest=./web -f

package main

import (
	"os"

	"github.com/audisto/data-downloader/cmd"
)

func main() {

	if err := cmd.RootCmd.Execute(); err != nil {
		cmd.PrintRed(err.Error())
		os.Exit(-1)
	}
}
