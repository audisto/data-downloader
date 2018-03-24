//go:generate statik -src=../../web/static -dest=../../web -f

package main

import (
	"os"
)

func main() {

	if err := RootCmd.Execute(); err != nil {
		PrintRed(err.Error())
		os.Exit(-1)
	}
}
