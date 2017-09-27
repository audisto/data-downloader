package cmd

import (
	"github.com/spf13/cobra"
)

// Command Line flags
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
	mode        string // pages or links
)

func registerPersistentFlags(rootCmd *cobra.Command) {
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&username, "username", "u", "", "Audisto API Username (required)")
	pf.StringVarP(&password, "password", "p", "", "Audisto API Password (required)")
	pf.Uint64VarP(&crawlID, "crawl", "c", 0, "ID of the crawl to download (required)")
	pf.StringVarP(&mode, "mode", "m", "pages", "Download mode, set it to 'links' or 'pages' (default)")
	pf.BoolVarP(&noDetails, "no-details", "d", false, "If passed, details in API request is set to 0")
	pf.StringVarP(&output, "output", "o", "", "Path for the output file")
	pf.BoolVarP(&noResume, "no-resume", "r", false, "If passed, download starts again, else the download is resumed")
	pf.StringVarP(&filter, "filter", "f", "", "Filter all pages by some attributes")
	pf.StringVarP(&order, "order", "", "", "Order by some attributes")
}

func requiredFlagsPassed() bool {
	return username != "" && password != "" && crawlID != 0
}

// Beside parsing flags and auto-type inferring offered by Cobra package
// we check for our own flag validations as well
func customFlagsValidation(cmd *cobra.Command) error {

	if !requiredFlagsPassed() {
		return CError("--username, --password and --crawl are required")
	}

	if username == "" {
		return CError("--username is required")
	}

	if password == "" {
		return CError("--password is required")
	}

	if crawlID == 0 {
		return CError("You need to also pass--crawl=NUMBER")
	}

	// validate mode
	if mode != "" && mode != "pages" && mode != "links" {
		msg := "mode has to be 'links' or 'pages', if this flag is dropped, it will default to 'pages'"
		return CError(msg)
	}
	return nil

}
