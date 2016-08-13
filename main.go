package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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
	res          Resumer
	outputWriter io.WriteCloser

	resumerSuffix string = ".audisto_"
)

type Resumer struct {
	OutputFilename string
	OutputPath     string // path minus filename

	DoneElements  int64
	TotalElements int64
	ChunkSize     int64

	httpClient http.Client
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

	// outputWriter.Write([]byte("hi there"))

	failCount := 0
	timeoutCount := 0

	for {
		chunk, statusCode, total, skip, err := res.nextChunk()
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 1)
			continue
		}

		// check status code
		switch {
		case statusCode == 429:
			{
				time.Sleep(time.Second * 30)
				break
			}
		case statusCode >= 400 && statusCode < 500:
			{
				switch statusCode {
				case 403:
					{
						fmt.Println("Access denied. Wrong credentials?")
						return
					}
				case 404:
					{
						fmt.Println("Not found. Correct crawl ID?")
						return
					}
				case 404:
					{
						fmt.Println("Unknown error occured")
						return
					}

				}

			}
		case statusCode == 504:
			{
				timeoutCount += 1
				if timeoutCount > 3 {
					if (res.ChunkSize - 10) > 0 {
						res.ChunkSize = res.ChunkSize - 10
					}
				}
				time.Sleep(time.Second * 30)
				break
			}
		case statusCode >= 500 && statusCode < 600:
			{
				time.Sleep(time.Second * 30)
				break
			}
		}

		scanner := bufio.NewScanner(bytes.NewReader(chunk))

		// is DoneElements == 0, don't skip first line
		if res.DoneElements == 0 {
			scanner.Scan()
			outputWriter.Write(scanner.Bytes())
		}

		// skip lines
		for i := int64(0); i < skip; i++ {
			scanner.Scan()
			fmt.Println("skipped this row: ", scanner.Text())
		}

		for scanner.Scan() {
			outputWriter.Write(scanner.Bytes())
		}

		if err := scanner.Err(); err != nil {
			fmt.Println(err)
			return
		}

	}
}

func (r *Resumer) nextChunkNumber() (nextChunkNumber, skipNRows int64) {

	if r.DoneElements == 0 {
		nextChunkNumber = 0
		skipNRows = 0
		return
	}

	skipNRows = r.DoneElements % r.ChunkSize
	nextChunkNumberFloat, _ := math.Modf(float64(r.DoneElements) / float64(r.ChunkSize))

	nextChunkNumber = int64(nextChunkNumberFloat)
	return
}

func (r *Resumer) nextChunk() ([]byte, int, int64, int64, error) {

	nextChunkNumber, skipNRows := r.nextChunkNumber()

	if r.DoneElements > 0 {
		skipNRows += 1
	}

	path := fmt.Sprintf("/2.0/crawls/%v/pages", crawl)
	method := "GET"

	headers := http.Header{}
	bodyParameters := url.Values{}

	queryParameters := url.Values{}
	if noDetails {
		queryParameters.Add("deep", "1")
	}
	queryParameters.Add("chunk", strconv.FormatInt(nextChunkNumber, 10))
	queryParameters.Add("chunk_size", strconv.FormatInt(r.ChunkSize, 10))
	queryParameters.Add("output", "tsv")

	body, statusCode, err := r.fetchRawChunk(path, method, headers, queryParameters, bodyParameters)
	if err != nil {
		return []byte(""), 0, 0, 0, err
	}

	bodyReader := bytes.NewReader(body)
	totalLines, err := lineCounter(bodyReader)
	if err != nil {
		return []byte(""), 0, 0, 0, err
	}

	return body, statusCode, int64(totalLines), skipNRows, nil
}

func lineCounter(r io.Reader) (int, error) {

	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

func (r *Resumer) fetchRawChunk(path string, method string, headers http.Header, queryParameters url.Values, bodyParameters url.Values) ([]byte, int, error) {

	domain := fmt.Sprintf("https://%s:%s@api.audisto.com", username, password)
	requestURL, err := url.Parse(domain)
	if err != nil {
		return []byte(""), 0, err
	}
	requestURL.Path = path
	requestURL.RawQuery = queryParameters.Encode()

	if method != "GET" && method != "POST" && method != "PATCH" && method != "DELETE" {
		return []byte(""), 0, fmt.Errorf("Method not supported: %v", method)
	}

	request, err := http.NewRequest(method, requestURL.String(), bytes.NewBufferString(bodyParameters.Encode()))
	if err != nil {
		return []byte(""), 0, fmt.Errorf("Failed to get the URL %s: %s", requestURL, err)
	}
	request.Header = headers
	request.Header.Add("Content-Length", strconv.Itoa(len(bodyParameters.Encode())))

	request.Header.Add("Connection", "Keep-Alive")
	request.Header.Add("Accept-Encoding", "gzip, deflate")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	response, err := r.httpClient.Do(request)
	if err != nil {
		return []byte(""), 0, fmt.Errorf("Failed to get the URL %s: %s", requestURL, err)
	}

	defer response.Body.Close()

	var responseReader io.ReadCloser
	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		decompressedBodyReader, err := gzip.NewReader(response.Body)
		if err != nil {
			return []byte(""), response.StatusCode, err
		}
		responseReader = decompressedBodyReader
		defer responseReader.Close()
	default:
		responseReader = response.Body
	}

	responseBody, err := ioutil.ReadAll(responseReader)
	if err != nil {
		return []byte(""), response.StatusCode, err
	}

	return responseBody, response.StatusCode, nil
}
