package downloader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strings"
	"time"
	// for debug purposes
)

const (
	// SMOOTHINGFACTOR -
	SMOOTHINGFACTOR = 0.005
)

var debugging = false // if true, debug messages will be shown

var (
	client       AudistoAPIClient
	downloader   Downloader
	outputWriter *bufio.Writer

	resumerSuffix = ".audisto_"

	output   string
	noResume bool
)

// Downloader initiate or resume a persisted downloading process info using AudistoAPIClient
// This also follows and increments chunk number, considering total elements to be downloaded
type Downloader struct {
	OutputFilename string `json:"outputFilename"`
	DoneElements   uint64 `json:"doneElements"`
	TotalElements  uint64 `json:"totalElements"`
	NoDetails      bool   `json:"noDetails"`
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

		downloader = Downloader{}

		downloader.TotalElements, err = client.GetTotalElements()
		if err != nil {
			return err
		}

		downloader.OutputFilename = output
		downloader.NoDetails = noDetails
	} else {

		errOutput, errResumer := fExists(output), fExists(output+resumerSuffix)
		startAnew := errOutput != nil && errResumer != nil

		// if don't resume, create new set
		if noResume || startAnew {

			if startAnew && !noResume {
				fmt.Println("No download to resume; starting a new...")
			}

			var err error

			downloader = Downloader{}

			downloader.TotalElements, err = client.GetTotalElements()
			if err != nil {
				return err
			}
			downloader.OutputFilename = output
			downloader.NoDetails = noDetails

			err = downloader.PersistConfig()
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
			err = json.Unmarshal(resumerFile, &downloader)
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

			if downloader.NoDetails != noDetails {
				return fmt.Errorf("this file was begun with --no-details=%v; continuing with --no-details=%v will break the file", downloader.NoDetails, noDetails)
			}

		}

	}

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

	return Run()
}

// Run runs the program
func Run() error {

	// only show progress bar when downloading to file
	if output != "" {
		go progressLoop()
	}

	debug(client.Username, client.Password, client.CrawlID)
	debugf("%#v\n", downloader)

MainLoop:
	for {
		var startTime time.Time = time.Now()
		var processedLines int64

		// res.chunkSize = int64(random(1000, 10000)) // debug; random chunk size

		progressPerc := downloader.progress()

		debugf("Progress: %.1f %%", progressPerc)

		// check if done
		if downloader.DoneElements == downloader.TotalElements {

			debug("@@@ COMPLETED 100% @@@")
			debugf("removing %v", output+resumerSuffix)

			// allow enought time for the progress bar to display
			// the "complete" message
			time.Sleep(time.Second)

			// when done, remove the resumer file
			if output != "" {
				os.Remove(output + resumerSuffix)
			}

			// exit program
			return nil
		}

		debugf("Calling next chunk")
		var chunk []byte
		var statusCode int
		var skip uint64
		err := retry(5, 10, func() error {
			var err error
			chunk, statusCode, skip, err = downloader.nextChunk()
			return err
		})

		if err != nil {
			debugf("Too many failures while calling next chunk; %v\n", err)
			return fmt.Errorf("Network error; please check your connection to the internet and resume download")
		}
		debugf("Next chunk obtained")
		debugf("statusCode: %v", statusCode)

		// if statusCode is not 200, up by one the error count
		// which is displayed in the progress bar
		if statusCode != 200 {
			errorCount++
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
						return fmt.Errorf("Wrong credentials")
					}
				case 403:
					{
						return fmt.Errorf("Access denied. Wrong credentials?")
					}
				case 404:
					{
						return fmt.Errorf("Not found. Correct crawl ID?")
					}
				default:
					{
						return fmt.Errorf("\nUnknown error occured (code %v)", statusCode)
					}
				}
			}
		case statusCode == 504:
			{
				timeoutCount++
				if timeoutCount >= 3 {
					// throttle
					if (client.ChunkSize - 1000) > 0 {

						// if chunkSize is 10000, throttle it down to 7000
						if client.ChunkSize == 10000 {
							client.ChunkSize -= 3000
						} else {
							// otherwise throttle it down by 1000
							client.ChunkSize -= 1000
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
		if downloader.DoneElements == 0 {
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
			downloader.DoneElements++

			// update the count of lines processed for this chunk
			processedLines++
		}

		// finalize every write
		outputWriter.Flush()

		// save to file the resumer data (to be able to resume later)
		downloader.PersistConfig()
		debugf("res.DoneElements = %v", downloader.DoneElements)

		// calculate average speed
		itTook := time.Since(startTime)
		temp := big.NewFloat(0).Quo(big.NewFloat(itTook.Seconds()), big.NewFloat(0).Quo(big.NewFloat(0).SetInt(big.NewInt(processedLines)), big.NewFloat(1000)))
		lastSpeed, _ := temp.Float64()
		averageSpeed := big.NewFloat(0).Add(big.NewFloat(0).Mul(big.NewFloat(SMOOTHINGFACTOR), big.NewFloat(lastSpeed)), big.NewFloat(0).Mul(big.NewFloat(0).Sub(big.NewFloat(0).SetInt(big.NewInt(1)), big.NewFloat(SMOOTHINGFACTOR)), big.NewFloat(averageTimePer1000)))
		averageTimePer1000, _ = averageSpeed.Float64()

		// scanner error
		if err := scanner.Err(); err != nil {
			errorCount++
			return fmt.Errorf("error wrile scanning chunk: %s", err.Error())
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
func (r *Downloader) nextChunkNumber() (nextChunkNumber, skipNRows uint64) {

	// if the remaining elements are less than the page size,
	// request only the remaining elements without having
	// to discard anything.
	remainingElements := r.TotalElements - r.DoneElements
	if remainingElements < client.ChunkSize && remainingElements > 0 {
		// r.chunkSize = remainingElements
		client.SetChunkSize(remainingElements)
	}

	// if no elements has been downloaded,
	// request the first chunk without skipping rows
	if r.DoneElements == 0 {
		nextChunkNumber = 0
		skipNRows = 0
		client.SetNextChunkNumber(0)
		return
	}

	// just in case
	if client.ChunkSize < 1 {
		// r.chunkSize = 1
		client.SetChunkSize(1)
	}

	skipNRows = r.DoneElements % client.ChunkSize
	nextChunkNumberFloat, _ := math.Modf(float64(r.DoneElements) / float64(client.ChunkSize))

	// just in case nextChunkNumber() gets called when all elements are
	// already downloaded, download chunk and discard all elements
	if r.DoneElements == r.TotalElements {
		skipNRows = 1
		// r.chunkSize = 1
		client.SetChunkSize(1)
	}

	nextChunkNumber = uint64(nextChunkNumberFloat)
	client.SetNextChunkNumber(nextChunkNumber)
	return
}

// nextChunk configures the API request and returns the chunk
func (r *Downloader) nextChunk() ([]byte, int, uint64, error) {

	_, skipNRows := r.nextChunkNumber()

	if r.DoneElements > 0 {
		skipNRows++
	}

	body, statusCode, err := client.FetchRawChunk(false)
	if err != nil {
		return []byte(""), 0, 0, err
	}

	return body, statusCode, skipNRows, nil
}

// PersistConfig saves the resumer to file
func (r *Downloader) PersistConfig() error {
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
