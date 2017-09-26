package cmd

import (
	"github.com/spf13/cobra"
)

var (
	username    string // Username for Audisto API authentication
	password    string // Password for audisto API authentication
	crawlID     uint64 // ID of the crawl to download
	chunkNumber uint64 // Number of Chunk
	chunkSize   uint64 // Elements in each chunk
	output      string // Output format
	filter      string // Possible filter
	noResume    bool   // Resume or not any previously downloaded file
	noDetails   bool   //  Request or not details from Audisto API
	order       string // Possible order of results
)

func registerPersistentFlags(rootCmd *cobra.Command) {
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&username, "username", "u", "", "Audisto API Username (required)")
	pf.StringVarP(&password, "password", "p", "", "Audisto API Password (required)")
	pf.Uint64VarP(&crawlID, "crawl", "c", 0, "ID of the crawl to download (required)")
	pf.BoolVarP(&noDetails, "no-details", "d", false, "If passed, details in API request is set to 0")
	pf.StringVarP(&output, "output", "o", "", "Path for the output file")
	pf.BoolVarP(&noResume, "no-resume", "r", false, "If passed, download starts again, else the download is resumed")
	pf.StringVarP(&filter, "filter", "f", "", "Filter all pages by some attributes")
	pf.StringVarP(&order, "order", "", "", "Order by some attributes")
}

func requiredFlagsPassed() bool {
	return username != "" && password != "" && crawlID != 0
}
