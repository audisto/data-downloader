package cmd

import (
	"time"

	"github.com/audisto/data-downloader/downloader"
	"github.com/spf13/cobra"
)

// RootCmd the root (default) command to data-downloader
var RootCmd = &cobra.Command{
	Use:   "data-downloader",
	Short: "Audisto Data Downloader",
	Long:  "A simple CLI tool to download data using Audisto API",
	// disable Cobra flags parsing we'll call our custom parse ourselves
	// to support the one-dash non-shorthand flags
	DisableFlagParsing: true,
	Example:            getExamples(),
	RunE: func(cmd *cobra.Command, args []string) error {
		// run our custom flags parsing
		err := customFlagsParse(cmd, args)
		if err != nil {
			return err
		}
		// Run our custom flags [values] validation
		err = customFlagsValidation(cmd)
		if err != nil {
			return err
		}

		// all looks good, perform the download
		return performDownload()
	},
}

func init() {
	// eatly register global flags that apply to the root command
	registerPersistentFlags(RootCmd)
}

// use Audisto downloader package to initiate/resume API downloads
func performDownload() error {
	progressReport := make(chan downloader.StatusReport)
	download := downloader.New(progressReport)

	err := download.Setup(username, password, crawlID, mode, noDetails,
		chunkNumber, chunkSize, output, filter, noResume, order, targets)

	if err != nil {
		return err
	}

	go RenderProgress(progressReport)

	err = download.Start()
	if err != nil {
		return err
	}
	// Give the progress bar a small time in order to refresh its rendering
	time.Sleep(time.Millisecond * 100)
	return nil
}
