package cmd

import (
	"fmt"
	"log"

	"github.com/audisto/data-downloader/downloader"
	"github.com/spf13/cobra"
)

// RootCmd the root (default) command to data-downloader
var RootCmd = &cobra.Command{
	Use:   "data-downloader",
	Short: "Audisto Data Downloader",
	Long:  "A simple CLI tool to download data using Audisto API",
	Run: func(cmd *cobra.Command, args []string) {

		if !requiredFlagsPassed() {
			cmd.Help()
		} else {
			fmt.Println("flags passed")
			performDownload()
		}
	},
	Example: getExamples(),
}

func init() {
	registerPersistentFlags(RootCmd)
}

func performDownload() {
	err := downloader.Get(username, password, crawlID, noDetails, chunkNumber,
		chunkSize, output, filter, noResume, order)

	if err != nil {
		log.Fatal(err)
	}
}

func getExamples() string {
	return `
$ data-downloader --username="USERNAME" --password="PASSWORD" --crawl=12345 --output="myCrawl.tsv"
$ data-downloader -u="USERNAME" -p="PASSWORD" -c=12345 -o="myCrawl.tsv" --no-resume
`
}
