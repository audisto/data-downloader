package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

var (
	outputWriter io.WriteCloser

	resumerSuffix string = ".audisto_"
)

type Resumer struct {
	OutputFilename string
	OutputPath     string // path minus filename

	Crawl     uint64
	NoDetails bool

	LatestDown string // filename of latest temp file downloaded

}

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
	output = strings.TrimSpace(output)

	// Check for non-valid flags
	usernameIsNull := username == ""
	passwordIsNull := password == ""
	crawlIsNull := crawl == 0

	if usernameIsNull || passwordIsNull || crawlIsNull {
		usage()
		os.Exit(0)
	}

	// stdout or output file ?
	if output == "" {
		outputWriter = os.Stdout
	} else {
		// If don't resume, create new set
		if noResume {
			var err error

			newConfig, err := json.Marshal(Resumer{})
			if err != nil {
				panic(err)
			}

			// create {{output}}.audisto_ file (keeps track of progress etc.)
			err = ioutil.WriteFile(output+resumerSuffix, newConfig, 0644)
			if err != nil {
				panic(err)
			}

			// create new outputFile
			outputWriter, err = os.Create(output)
			if err != nil {
				panic(err)
			}
		}

		// if resume, check if output file exists
		if err := fExists(output); err != nil {
			panic(fmt.Sprint("Cannot resume; output file does not exist: ", err))
		}
		// if resume, check if resume file exists
		if err := fExists(output + resumerSuffix); err != nil {
			panic(fmt.Sprint("Cannot resume; resume file does not exist: ", err))
		}

		// read and validate resumer file
		// read and validate output file

	}

}

// fExists returns nil if path is an existing file/folder
func fExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}
	return nil
}

func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	return fileInfo.IsDir(), err
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

	outputWriter.Write([]byte("hi there"))
}
