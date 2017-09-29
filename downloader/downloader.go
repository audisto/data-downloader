package downloader

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gosuri/uilive"
	// for debug purposes
)

var debugging = false // if true, debug messages will be shown

var (
	client       AudistoAPIClient
	res          Resumer
	outputWriter *bufio.Writer

	resumerSuffix = ".audisto_"

	output   string
	noResume bool
)

// progress bar elements
var (
	progressIndicator *uilive.Writer
	progressStatus    string

	timeoutCount int
	errorCount   int

	averageTimePer1000 float64 = 1
)

type Resumer struct {
	OutputFilename string `json:"outputFilename"`

	chunkSize     uint64
	DoneElements  uint64 `json:"doneElements"`
	TotalElements uint64 `json:"totalElements"`
	NoDetails     bool   `json:"noDetails"`

	httpClient http.Client
}

// chunk is used to get unmarshal the json containing the total number of chunks
type chunk struct {
	Chunk struct {
		Total uint64 `json:"total"`
		Page  int    `json:"page"`
		Size  int    `json:"size"`
	} `json:"chunk"`
}

// Initialize assign parsed flags or params to the local package variables
func Initialize(username string, password string, crawl uint64, mode string,
	noDetails bool, chunkNumber uint64, chunkSize uint64, fOutput string,
	filter string, fNoResume bool, order string) error {

	output = fOutput
	noResume = fNoResume

	client = AudistoAPIClient{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(password),
		CrawlID:  crawl,
		Mode:     strings.TrimSpace(mode),
		Deep:     noDetails != true,
		Order:    strings.TrimSpace(order),
		Filter:   strings.TrimSpace(filter),
	}

	client.SetChunkSize(chunkSize)

	if err := client.IsValid(); err != nil {
		return err
	}

	// stdout or output file ?
	if output == "" {
		outputWriter = bufio.NewWriter(os.Stdout)

		var err error

		res = Resumer{}

		res.TotalElements, err = totalElements()
		if err != nil {
			return err
		}

		res.OutputFilename = output
		res.NoDetails = noDetails
	} else {

		errOutput, errResumer := fExists(output), fExists(output+resumerSuffix)
		startAnew := errOutput != nil && errResumer != nil

		// if don't resume, create new set
		if noResume || startAnew {

			if startAnew && !noResume {
				fmt.Println("No download to resume; starting a new...")
			}

			var err error

			res = Resumer{}

			res.TotalElements, err = totalElements()
			if err != nil {
				fmt.Println(err)
				os.Exit(0)
			}
			res.OutputFilename = output
			res.NoDetails = noDetails

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
			if errOutput != nil {
				return fmt.Errorf("cannot resume; %q file does not exist: use --no-resume to create new", output)
			}
			// if resume, check if resume file exists
			if errResumer != nil {
				return fmt.Errorf("cannot resume; resumer file %v does not exist", output+resumerSuffix)
			}

			resumerFile, err := ioutil.ReadFile(output + resumerSuffix)
			if err != nil {
				panic(fmt.Sprintf("Resumer file error: %v\n", err))
			}
			err = json.Unmarshal(resumerFile, &res)
			if err != nil {
				panic(fmt.Sprintf("Resumer file error: %v\n", err))
			}

			// open outputFile
			existingFile, err := os.OpenFile(output, os.O_WRONLY|os.O_APPEND, 0777)
			if err != nil {
				panic(err)
			}
			outputWriter = bufio.NewWriter(existingFile)

			// read and validate resumer file
			// read and validate output file
			// check last id of the last write batch

			if res.NoDetails != noDetails {
				return fmt.Errorf("this file was begun with --no-details=%v; continuing with --no-details=%v will break the file", res.NoDetails, noDetails)
			}

		}

	}

	res.chunkSize = client.ChunkSize
	return nil
}

// Get assign params and execute the Run() function
func Get(username string, password string, crawl uint64, mode string,
	deep bool, chunknumber uint64, chunkSize uint64, output string,
	filter string, noResume bool, order string) error {

	err := Initialize(username, password, crawl, mode, deep, chunknumber, chunkSize,
		output, filter, noResume, order)

	if err != nil {
		return err
	}

	Run()
	return nil
}

// Run runs the program
func Run() {

	// only show progress bar when downloading to file
	if output != "" {
		progressIndicator = uilive.New()
		progressIndicator.Start()
		go progressLoop()
	}

	debug(client.Username, client.Password, client.CrawlID)
	debugf("%#v\n", res)

MainLoop:
	for {
		var startTime time.Time = time.Now()
		var processedLines int64

		// res.chunkSize = int64(random(1000, 10000)) // debug; random chunk size

		progressPerc := res.progress()
		updateStatus(fmt.Sprintf("%.1f%% of %v %s", progressPerc, res.TotalElements, client.Mode))
		debugf("Progress: %.1f %%", progressPerc)

		// check if done
		if res.DoneElements == res.TotalElements {
			updateStatus("@@@ COMPLETED 100% @@@")

			debug("@@@ COMPLETED 100% @@@")
			debugf("removing %v", output+resumerSuffix)

			// allow enought time for the progress bar to display
			// the "complete" message
			time.Sleep(time.Second)

			// when done, remove the resumer file
			if output != "" {
				os.Remove(output + resumerSuffix)

				// stop the progress bar
				progressIndicator.Stop()
			}

			// exit program
			return
		}

		debugf("Calling next chunk")
		var chunk []byte
		var statusCode int
		var skip uint64
		err := retry(5, 10, func() error {
			var err error
			chunk, statusCode, skip, err = res.nextChunk()
			return err
		})

		if err != nil {
			debugf("Too many failures while calling next chunk; %v\n", err)
			fmt.Println("Network error; please check your connection to the internet and resume download.")
			return
		}
		debugf("Next chunk obtained")
		debugf("statusCode: %v", statusCode)

		// if statusCode is not 200, up by one the error count
		// which is displayed in the progress bar
		if statusCode != 200 {
			errorCount += 1
		}

		// check status code

		switch {
		case statusCode == 429:
			{
				// meaning: multiple requests
				time.Sleep(time.Second * 30)
				continue MainLoop
			}
		case statusCode >= 400 && statusCode < 500:
			{
				switch statusCode {
				case 401:
					{
						fmt.Println("Wrong credentials.")
						return
					}
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
						fmt.Printf("\nUnknown error occured (code %v).\n", statusCode)
						return
					}
				}
			}
		case statusCode == 504:
			{
				timeoutCount += 1
				if timeoutCount >= 3 {
					// throttle
					if (res.chunkSize - 1000) > 0 {

						// if chunkSize is 10000, throttle it down to 7000
						if res.chunkSize == 10000 {
							res.chunkSize -= 3000
						} else {
							// otherwise throttle it down by 1000
							res.chunkSize -= 1000
						}

						// reset the timeout count
						timeoutCount = 0
					}
				}
				time.Sleep(time.Second * 30)
				continue MainLoop
			}
		case statusCode >= 500 && statusCode < 600:
			{
				// meaning: server error
				time.Sleep(time.Second * 30)
				continue MainLoop
			}
		}

		if statusCode != 200 {
			// just in case it's not an error in the ranges above
			continue MainLoop
		}

		// iterator for the received chunk
		scanner := bufio.NewScanner(bytes.NewReader(chunk))
		debugf("chunk bytes len: %v", len(chunk))

		// write the header of the tsv
		if res.DoneElements == 0 {
			scanner.Scan()
			outputWriter.Write(append(scanner.Bytes(), []byte("\n")...))
		}

		// skip lines that we alredy have
		for i := uint64(0); i < skip; i++ {
			scanner.Scan()
			debugf("skipping this row: \n%s ", scanner.Text())
		}

		// iterate over the remaining lines
		for scanner.Scan() {
			// write lines (to stdout or file)
			outputWriter.Write(append(scanner.Bytes(), []byte("\n")...))

			// update the in-memory resumer
			res.DoneElements++

			// update the count of lines processed for this chunk
			processedLines++
		}

		// finalize every write
		outputWriter.Flush()

		// save to file the resumer data (to be able to resume later)
		res.PersistConfig()
		debugf("res.DoneElements = %v", res.DoneElements)

		// calculate average speed
		itTook := time.Since(startTime)
		temp := big.NewFloat(0).Quo(big.NewFloat(itTook.Seconds()), big.NewFloat(0).Quo(big.NewFloat(0).SetInt(big.NewInt(processedLines)), big.NewFloat(1000)))
		lastSpeed, _ := temp.Float64()
		SMOOTHING_FACTOR := 0.005
		averageSpeed := big.NewFloat(0).Add(big.NewFloat(0).Mul(big.NewFloat(SMOOTHING_FACTOR), big.NewFloat(lastSpeed)), big.NewFloat(0).Mul(big.NewFloat(0).Sub(big.NewFloat(0).SetInt(big.NewInt(1)), big.NewFloat(SMOOTHING_FACTOR)), big.NewFloat(averageTimePer1000)))
		averageTimePer1000, _ = averageSpeed.Float64()

		// scanner error
		if err := scanner.Err(); err != nil {
			errorCount += 1
			fmt.Println("error wrile scanning chunk: ", err)
			return
		}

	}
}

// retry operation
func retry(attempts int, sleep int, callback func() error) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return nil
		}

		if i >= (attempts - 1) {
			break
		}

		errorCount++

		// pause before retrying
		time.Sleep(time.Duration(sleep) * time.Second)

		debugf("Something failed, retrying;")
	}
	return fmt.Errorf("Abandoned after %d attempts, last error: %s", attempts, err)
}

// nextChunkNumber calculates the index of the next chunk,
// and also returns the number of rows to skip.
// nextChunkNumber is used to calculate the next chunk number after resuming
// and also to recalculate the chunk number in case of throttling.
func (r *Resumer) nextChunkNumber() (nextChunkNumber, skipNRows uint64) {

	// if the remaining elements are less than the page size,
	// request only the remaining elements without having
	// to discard anything.
	remainingElements := r.TotalElements - r.DoneElements
	if remainingElements < r.chunkSize &&
		remainingElements > 0 {
		r.chunkSize = remainingElements
	}

	// if no elements has been downloaded,
	// request the first chunk without skipping rows
	if r.DoneElements == 0 {
		nextChunkNumber = 0
		skipNRows = 0
		return
	}

	// just in case
	if r.chunkSize < 1 {
		r.chunkSize = 1
	}

	skipNRows = r.DoneElements % r.chunkSize
	nextChunkNumberFloat, _ := math.Modf(float64(r.DoneElements) / float64(r.chunkSize))

	// just in case nextChunkNumber() gets called when all elements are
	// already downloaded, download chunk and discard all elements
	if r.DoneElements == r.TotalElements {
		skipNRows = 1
		r.chunkSize = 1
	}

	nextChunkNumber = uint64(nextChunkNumberFloat)
	return
}

// nextChunk configures the API request and returns the chunk
func (r *Resumer) nextChunk() ([]byte, int, uint64, error) {

	nextChunkNumber, skipNRows := r.nextChunkNumber()

	if r.DoneElements > 0 {
		skipNRows++
	}

	path := fmt.Sprintf("/2.0/crawls/%v/%s", client.CrawlID, client.Mode)
	method := "GET"

	headers := http.Header{}
	bodyParameters := url.Values{}

	queryParameters := url.Values{}
	if client.Deep {
		queryParameters.Add("deep", "1")
	} else {
		queryParameters.Add("deep", "0")
	}
	if client.Filter != "" {
		queryParameters.Add("filter", client.Filter)
	}
	if client.Order != "" {
		queryParameters.Add("order", client.Order)
	}
	queryParameters.Add("chunk", strconv.FormatUint(nextChunkNumber, 10))
	queryParameters.Add("chunk_size", strconv.FormatUint(r.chunkSize, 10))
	queryParameters.Add("output", "tsv")

	body, statusCode, err := r.fetchRawChunk(path, method, headers, queryParameters, bodyParameters)
	if err != nil {
		return []byte(""), 0, 0, err
	}

	return body, statusCode, skipNRows, nil
}

// fetchRawChunk makes the request to the server for a chunk
func (r *Resumer) fetchRawChunk(path string, method string, headers http.Header, queryParameters url.Values, bodyParameters url.Values) ([]byte, int, error) {

	domain := fmt.Sprintf("https://%s:%s@api.audisto.com", client.Username, client.Password)
	debug(domain)
	debug(client.Username)
	debug(client.Password)
	requestURL, err := url.Parse(domain)
	if err != nil {
		return []byte(""), 0, err
	}
	requestURL.Path = path
	requestURL.RawQuery = queryParameters.Encode()

	if method != "GET" && method != "POST" && method != "PATCH" && method != "DELETE" {
		return []byte(""), 0, fmt.Errorf("Method not supported: %v", method)
	}

	debugf("request url: %s", requestURL.String())
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

// totalElements asks the server the total number of elements
func totalElements() (uint64, error) {
	var body []byte
	var statusCode int

	err := retry(5, 3, func() error {
		var err error
		body, statusCode, err = res.fetchTotalElements()
		if err != nil {
			return err
		}

		switch {
		case statusCode == 429:
			{
				// meaning: multiple requests
				err = fmt.Errorf("Error while getting total number of elements: 429, multiple requests")
				time.Sleep(time.Second * 5)
			}
		case statusCode >= 400 && statusCode < 500:
			{
				switch statusCode {
				case 401:
					{
						fmt.Println("Wrong credentials.")
						os.Exit(0)
					}
				case 403:
					{
						fmt.Println("Access denied. Wrong credentials?")
						os.Exit(0)
					}
				case 404:
					{
						fmt.Println("Not found. Correct crawl ID?")
						os.Exit(0)
					}
				default:
					{
						fmt.Printf("\nUnknown error occured (code %v).\n", statusCode)
						os.Exit(0)
					}
				}
			}
		case statusCode == 504:
			{
				// meaning: timeout
				err = fmt.Errorf("Error while getting total number of elements: 504, server timeout")
				time.Sleep(time.Second * 5)
			}
		case statusCode >= 500 && statusCode < 600:
			{
				// meaning: server error
				err = fmt.Errorf("Error while getting total number of elements: %v, server error", statusCode)
				time.Sleep(time.Second * 5)
			}
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

// fetchTotalElements sets up the request for the first chunk in json,
// containing the total number of elements.
func (r *Resumer) fetchTotalElements() ([]byte, int, error) {

	path := fmt.Sprintf("/2.0/crawls/%v/%s", client.CrawlID, client.Mode)
	method := "GET"

	headers := http.Header{}
	bodyParameters := url.Values{}

	queryParameters := url.Values{}
	queryParameters.Add("deep", "0")
	queryParameters.Add("chunk", "0")
	queryParameters.Add("chunk_size", "1")
	queryParameters.Add("output", "json")
	if client.Filter != "" {
		queryParameters.Add("filter", client.Filter)
	}
	if client.Order != "" {
		queryParameters.Add("order", client.Order)
	}

	body, statusCode, err := r.fetchRawChunk(path, method, headers, queryParameters, bodyParameters)
	if err != nil {
		return []byte(""), 0, err
	}

	return body, statusCode, nil
}

// PersistConfig saves the resumer to file
func (r *Resumer) PersistConfig() error {
	// save config to file only if not printing to stdout
	if output == "" {
		return nil
	}

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

// progress outputs the progress percentage
func (r *Resumer) progress() *big.Float {
	var progressPerc *big.Float = big.NewFloat(0.0)
	if res.TotalElements > 0 && res.DoneElements > 0 {
		progressPerc = big.NewFloat(0).Quo(big.NewFloat(100), big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(res.TotalElements), big.NewFloat(0).SetUint64(res.DoneElements)))
	}
	return progressPerc
}

// updateStatus sets the first part of the progress bar message
func updateStatus(s string) {
	// TODO: add a mutex?
	progressStatus = s
}

// progress animation
func progressLoop() {
	var n int = 0
	var max int = 10
	for {

		ETAuint64, _ := big.NewFloat(0).Quo(big.NewFloat(0).Quo(big.NewFloat(0).Sub(big.NewFloat(0).SetUint64(res.TotalElements), big.NewFloat(0).SetUint64(res.DoneElements)), big.NewFloat(1000)), big.NewFloat(averageTimePer1000)).Uint64()
		ETAtime := time.Duration(ETAuint64) * time.Millisecond * 110
		ETAstring := ETAtime.String()

		progressMessage := progressStatus + chs(n, ".") + chs(max-n, "*")
		progressMessage = progressMessage + fmt.Sprintf(" | ETA %v |", ETAstring)
		progressMessage = progressMessage + fmt.Sprintf(" Chunk size %v |", res.chunkSize)
		progressMessage = progressMessage + fmt.Sprintf(" %v timeouts |", timeoutCount)
		progressMessage = progressMessage + fmt.Sprintf(" %v errors |", errorCount)

		fmt.Fprintln(progressIndicator, progressMessage)
		time.Sleep(time.Millisecond * 500)

		n++
		if n >= max {
			n = 0
		}
	}
}
