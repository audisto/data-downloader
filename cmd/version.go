package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VERSION Audisto data downloader version number
const VERSION = "0.6"

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of data-downloader",
	Long:  `Print the version number of data-downloader`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("data-downloader v" + VERSION)
	},
}
