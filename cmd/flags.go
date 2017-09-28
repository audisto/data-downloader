package cmd

import (
	"fmt"
	"os"
	"strings"

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
	noDetails   bool   // Request or not details from Audisto API
	order       string // Possible order of results
	mode        string // pages or links
	targets     string // "self" or a path to a file containing link target pages (IDs)
)

// register global flags that apply to the root command
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
	pf.StringVarP(&targets, "targets", "t", "", `"self" or a path to a file containing link target pages (IDs)`)
}

// check if --username --password and --crawl are being passed with non-empty values
func requiredFlagsPassed() bool {
	return username != "" && password != "" && crawlID != 0
}

// Beside parsing flags and auto-type inferring offered by Cobra package
// we check for our own flag validations/logic as well
func customFlagsValidation(cmd *cobra.Command) error {

	// make sure required flags are passed
	if !requiredFlagsPassed() {
		return CError("--username, --password and --crawl are required")
	}

	// normalize flags before proceeding with the validation
	normalizeFlags()

	// validate mode
	if mode != "" && mode != "pages" && mode != "links" {
		msg := "mode has to be 'links' or 'pages', if this flag is dropped, it will default to 'pages'"
		return CError(msg)
	}

	// validate targets / mode / filter combinations
	if targets != "" {

		// do not allow --filter when --targets is being used
		if filter != "" {
			return CError("Set either --filter or --targets, but not both")
		}

		// --mode=pages is only allowed when targets=self
		if targets == "self" && mode != "pages" {
			return CError("Set --mode=pages to use --targets=self")
		}

		// --targets=FILEPATH is only allowed when mode is set to links
		// we'd also make sure the file exists.
		if targets != "self" {

			if mode != "links" {
				return CError("Set --mode=links to use --targets=FILEPATH")
			}

			if _, err := os.Stat(targets); os.IsNotExist(err) {
				return CError(fmt.Sprintf("%s file does not exist", targets))
			}
		}

	}
	// returning no error means the validation passed
	return nil
}

// trim spaces and lowercase [some] string-based flags
func normalizeFlags() {

	// trim spaces for 'mode', 'targets', 'output', 'filter' and 'order'
	mode = strings.TrimSpace(mode)
	targets = strings.TrimSpace(targets)
	output = strings.TrimSpace(output)
	filter = strings.TrimSpace(filter)
	order = strings.TrimSpace(order)

	// lowercase 'mode' and
	mode = strings.ToLower(mode)

	// lowercase 'targets' when it's being set to 'self'
	if strings.EqualFold(targets, "self") {
		targets = "self"
	}
}
