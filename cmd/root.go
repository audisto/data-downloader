package cmd

import (
	"github.com/audisto/data-downloader/downloader"
	"github.com/spf13/cobra"
)

// RootCmd the root (default) command to data-downloader
var RootCmd = &cobra.Command{
	Use:   "data-downloader",
	Short: "Audisto Data Downloader",
	Long:  "A simple CLI tool to download data using Audisto API",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Run our custom flags validation
		err := customFlagsValidation(cmd)
		if err != nil {
			return err
		}
		return performDownload()
	},
	Example: getExamples(),
}

func init() {
	// register global flags that applies to all commands.
	registerPersistentFlags(RootCmd)
}

// use Audisto downloader package to initiate/resume API downloads
func performDownload() error {
	return downloader.Get(username, password, crawlID, mode, noDetails,
		chunkNumber, chunkSize, output, filter, noResume, order)
}

// example command usage hooked into the CLI usage text.
func getExamples() string {
	// Todo: change color
	return StringYellow(`
$ data-downloader --username="USERNAME" --password="PASSWORD" --crawl=12345 --output="myCrawl.tsv"
$ data-downloader -u="USERNAME" -p="PASSWORD" -c=12345 -o="myCrawl.tsv" --no-resume -m=links
`)
}
