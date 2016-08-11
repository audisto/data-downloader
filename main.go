package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	username string
	password string
	crawl    uint64

	noDetails bool
	output    string
	noResume  bool
)

func init() {

	flag.StringVar(&username, "username", "", "API Username (required)")
	flag.StringVar(&password, "password", "", "API Password (required)")
	flag.Uint64Var(&crawl, "crawl", 0, "ID of the crawl to download (required)")

	flag.BoolVar(&noDetails, "no-details", false, "If passed, details in API request is set to 0 else")
	flag.StringVar(&output, "output", "", "Path for the output file")
	flag.BoolVar(&noResume, "no-resume", false, "If passed, download starts again, else the download is resumed")

	flag.Usage = usage
	flag.Parse()

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	// Check for empty flags
	usernameIsNull := username == ""
	passwordIsNull := password == ""
	crawlIsNull := crawl == 0

	if usernameIsNull || passwordIsNull || crawlIsNull {
		usage()
		os.Exit(0)
	}

}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: audistoDownloader [flags]
Flags:
  username    API Username (required)
  password    API Password (required)
  crawl       ID of the crawl to download (required)
  no-details  If passed, details in API request is set to 0 else
  output      Path for the output file
  no-resume   If passed, download starts again, else the download is resumed
`)
}

func main() {
	fmt.Println(username, password, crawl)
}
