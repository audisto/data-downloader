package cmd

import (
	"github.com/audisto/data-downloader/web"
	"github.com/spf13/cobra"
)

var (
	port uint = 5050
)

func init() {
	RootCmd.AddCommand(webCmd)
	webCmd.Flags().UintVarP(&port, "port", "P", 5050, "Web server port (default is 5050)")
}

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch a local web interface of data-downloader",
	Long:  `Launch a local web interface of data-downloader`,
	RunE: func(cmd *cobra.Command, args []string) error {
		web.StartWebInterface(port, false)
		return nil
	},
}
