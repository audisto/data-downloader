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

var debugging = true

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
	outputWriter *bufio.Writer

	resumerSuffix string = ".audisto_"
)

type Resumer struct {
	OutputFilename string

	DoneElements  int64
	TotalElements int64
	chunkSize     int64

	httpClient http.Client
}

type chunk struct {
	Chunk struct {
		Total int64
		Page  int
		Size  int
	}
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
		outputWriter = bufio.NewWriter(os.Stdout)
	} else {
		// If don't resume, create new set
		if noResume {

			// if resume, check if output file exists
			if err := fExists(output); err == nil {
				panic("File already exists; please delete or specify another output filename")
			}

			var err error

			res = Resumer{}

			res.TotalElements, err = totalElements()
			if err != nil {
				panic(err)
			}
			res.OutputFilename = output

			err = res.PersistConfig()
			if err != nil {
				panic(err)
			}

			// create new outputFile
			newFile, err := os.Create(output)
			if err != nil {
				panic(err)
			}
			outputWriter = bufio.NewWriter(newFile)
		} else {
			// if resume, check if output file exists
			if err := fExists(output); err != nil {
				panic(fmt.Sprint("Cannot resume; output file does not exist: ", err))
			}
			// if resume, check if resume file exists
			if err := fExists(output + resumerSuffix); err != nil {
				panic(fmt.Sprint("Cannot resume; resume file does not exist: ", err))
			}

			resumerFile, err := ioutil.ReadFile(output + resumerSuffix)
			if err != nil {
				panic(fmt.Sprintf("File error: %v\n", err))
			}
			json.Unmarshal(resumerFile, &res)

			// open outputFile
			existingFile, err := os.OpenFile(output, os.O_WRONLY|os.O_APPEND, 0777)
			if err != nil {
				panic(err)
			}
			outputWriter = bufio.NewWriter(existingFile)

			// read and validate resumer file
			// read and validate output file

		}

	}

	res.chunkSize = 10000

}

func main() {
	debug(username, password, crawl)

	// outputWriter.Write([]byte("hi there"))

	timeoutCount := 0

	debug("%#v\n", res)
	for {
		debug("calling next chunk")
		chunk, statusCode, skip, err := res.nextChunk()
		if err != nil {
			debug("error while calling next chunk; %v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}
		debug("next chunk obtained")
		debug("statusCode: %v", statusCode)

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
				default:
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
					if (res.chunkSize - 10) > 0 {
						res.chunkSize = res.chunkSize - 10
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
		debug("chunk len: %v", len(chunk))

		// is DoneElements == 0, don't skip first line
		if res.DoneElements == 0 {
			scanner.Scan()
			outputWriter.Write(scanner.Bytes())
			outputWriter.Write([]byte("\n"))
		}

		// skip lines
		for i := int64(0); i < skip; i++ {
			scanner.Scan()
			debug("skipped this row: ", scanner.Text())
		}

		for scanner.Scan() {
			outputWriter.Write(scanner.Bytes())
			outputWriter.Write([]byte("\n"))
			res.DoneElements += 1
		}

		outputWriter.Flush()
		debug("res.DoneElements = %v", res.DoneElements)

		if err := scanner.Err(); err != nil {
			fmt.Println(err)
			return
		}

	}
}

func debug(format string, a ...interface{}) (n int, err error) {
	if debugging {
		return fmt.Printf("\n"+format+"\n", a...)
	}
	return 0, nil
}

func (r *Resumer) nextChunkNumber() (nextChunkNumber, skipNRows int64) {

	if r.DoneElements == 0 {
		nextChunkNumber = 0
		skipNRows = 0
		return
	}

	skipNRows = r.DoneElements % r.chunkSize
	nextChunkNumberFloat, _ := math.Modf(float64(r.DoneElements) / float64(r.chunkSize))

	nextChunkNumber = int64(nextChunkNumberFloat)
	return
}

func (r *Resumer) nextChunk() ([]byte, int, int64, error) {

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
		queryParameters.Add("deep", "0")
	} else {
		queryParameters.Add("deep", "1")
	}
	queryParameters.Add("chunk", strconv.FormatInt(nextChunkNumber, 10))
	queryParameters.Add("chunk_size", strconv.FormatInt(r.chunkSize, 10))
	queryParameters.Add("output", "tsv")

	body, statusCode, err := r.fetchRawChunk(path, method, headers, queryParameters, bodyParameters)
	if err != nil {
		return []byte(""), 0, 0, err
	}

	return body, statusCode, skipNRows, nil
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

	debug("request url: %s", requestURL.String())
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

func totalElements() (int64, error) {
	var body []byte
	var statusCode int
	err := retry(5, 1, func() error {
		var err error
		body, statusCode, err = res.fetchTotalElements()
		if err != nil {
			return err
		}
		if statusCode != 200 {
			err = fmt.Errorf("error while trying to get total number of elements; statusCode %v", statusCode)
		}
		return err
	})
	if err != nil {
		return 0, err
	}

	var firstChunk chunk
	err = json.Unmarshal(body, &firstChunk)
	if err != nil {
		return 0, err
	}

	return firstChunk.Chunk.Total, nil
}

func (r *Resumer) fetchTotalElements() ([]byte, int, error) {

	path := fmt.Sprintf("/2.0/crawls/%v/pages", crawl)
	method := "GET"

	headers := http.Header{}
	bodyParameters := url.Values{}

	queryParameters := url.Values{}
	queryParameters.Add("deep", "0")
	queryParameters.Add("chunk", "0")
	queryParameters.Add("chunk_size", "1")
	queryParameters.Add("output", "json")

	body, statusCode, err := r.fetchRawChunk(path, method, headers, queryParameters, bodyParameters)
	if err != nil {
		return []byte(""), 0, err
	}

	return body, statusCode, nil
}

// PersistConfig saves the accounts to file
func (r *Resumer) PersistConfig() error {
	config, err := json.MarshalIndent(r, "", "	")
	if err != nil {
		return err
	}

	// create {{output}}.audisto_ file (keeps track of progress etc.)
	err = ioutil.WriteFile(output+resumerSuffix, config, 0644)
	if err != nil {
		return err
	}
	return nil
}

func retry(attempts int, sleep int, callback func() error) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return nil
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(time.Duration(sleep) * time.Second)
		debug("Something failed, retrying;")
	}
	return fmt.Errorf("Abandoned after %d attempts, last error: %s", attempts, err)
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
