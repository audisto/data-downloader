package downloader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"
	// for debug purposes
)

const (
	// SMOOTHINGFACTOR -
	SMOOTHINGFACTOR = 0.005
	resumerSuffix   = ".audisto_"
)

var debugging = false // if true, debug messages will be shown

var (
	client       AudistoAPIClient
	downloader   Downloader
	outputWriter *bufio.Writer
)

// Downloader initiate or resume a persisted downloading process info using AudistoAPIClient
// This also follows and increments chunk number, considering total elements to be downloaded
type Downloader struct {
	OutputFilename    string        `json:"outputFilename"`
	TargetsFilename   string        `json:"targetsFilename"`
	DoneElements      uint64        `json:"doneElements"`
	TotalElements     uint64        `json:"totalElements"`
	NoDetails         bool          `json:"noDetails"`
	TargetsFileMD5    string        `json:"targetsFileMD5"`
	TargetsFileNextID int           `json:"targetsFileNextID"`
	CurrentTarget     currentTarget `json:"currentTarget"`

	noResume               bool
	currentTargetsFilename string
	currentTargetsMd5Hash  string
	ids                    []uint64
	elements               map[uint64]uint64 // [pageID] => totalElements
}

// current download target.
// in case of 'targets' mode, this will be dynamic
type currentTarget struct {
	DoneElements  uint64 `json:"doneElements"`
	TotalElements uint64 `json:"totalElements"`
}

func new(outputFilename string, noResume bool, targets string) (d Downloader) {
	return Downloader{
		OutputFilename:         strings.TrimSpace(outputFilename),
		noResume:               noResume,
		currentTargetsFilename: strings.TrimSpace(targets),
	}
}

// getResumeFilename construct the complete file name of the resume file
func (d *Downloader) getResumeFilename() string {
	return d.OutputFilename + resumerSuffix
}

// tryResume check to see if the current download can be a resume of a previous one
func (d *Downloader) tryResume(noDetails bool) (canBeResumed bool, err error) {

	// Are we outputing to some file?
	if d.OutputFilename == "" || d.noResume {
		return false, nil
	}

	resumeFileExists, outputFileExists := fExists(d.getResumeFilename()), fExists(d.OutputFilename)

	// check if we already have a complete download before?
	noNeedForResume := resumeFileExists != nil && outputFileExists == nil
	if noNeedForResume {
		err = fmt.Errorf("%q file seems already downloaded: use --no-resume to create new", d.OutputFilename)
		return false, err
	}

	// Does a resume meta info file exist?
	if resumeFileExists != nil {
		// do not return an error, just start anew
		return false, nil
	}

	// If we have an UNFINISHED or FRESH download..
	// Does the previous output file itself exist?
	if outputFileExists != nil {
		err = fmt.Errorf("cannot resume; %q file does not exist: use --no-resume to create new", d.OutputFilename)
		return false, err
	}

	// So far, it looks like there is a resume file, lets try opening it
	resumerFile, err := ioutil.ReadFile(d.getResumeFilename())
	if err != nil {
		return false, fmt.Errorf("resumer file error: %v", err)
	}

	// try to unmarshal the resumer file to the current downloader
	err = json.Unmarshal(resumerFile, &d)
	if err != nil {
		return false, fmt.Errorf("resumer file error: %v", err)
	}

	// Is there a conflict about whether or not details are to be downloaded
	if d.NoDetails != noDetails {
		err = fmt.Errorf("this file was begun with --no-details=%v; continuing with --no-details=%v will break the file", d.NoDetails, noDetails)
		return false, err
	}

	// So far, so good, but..
	// Are we in targets mode? if so, check if the previous targets filepath matches the new one
	// We need to ensure consistency, and that we're correctly following the line numbers of the same file
	if d.isInTargetsMode() {
		// did the user run the same command before but without --targets=File ?
		if d.TargetsFilename == "" {
			msg := "you are trying to resume a download that had no targets specified before\n"
			msg += "you need to explicitly pass '--no-resume' flag to start a new download"
			return false, fmt.Errorf(msg)
		}

		// did the user change the targets filename ?
		if d.currentTargetsFilename != d.TargetsFilename {
			msg := "this download was previously started with a different targets file.\n"
			msg += "previous target file: " + d.TargetsFilename + "\n"
			msg += "current target file: " + d.currentTargetsFilename + "\n"
			msg += "to ensure the resume from the previous line number, you need to specify the previous file as is"
			msg += " or pass a 'no-resume' flag to start anew"
			err = fmt.Errorf(msg)
			return false, err
		}

		// even if the filename is the same, calculate MD5 hash to ensure the content of the file did not change
		fileMD5, err := getFileMD5Hash(d.currentTargetsFilename)
		if err != nil {
			return false, err
		}

		if fileMD5 != d.TargetsFileMD5 {
			err = fmt.Errorf("targets file content has been altered, abording an inconsistent resume")
			return false, err
		}

	} else {
		if d.TargetsFilename != "" {
			msg := "you are trying to resume a download that had targets file specified before\n"
			msg += "you need to explicitly pass '--no-resume' flag to start a new download"
			return false, fmt.Errorf(msg)
		}
	}

	return true, nil
}

func (d *Downloader) isDone() bool {
	return d.CurrentTarget.DoneElements >= d.CurrentTarget.TotalElements
}

func (d *Downloader) processTargetsFile() error {
	ids, err := ProcessTargetFile(d.currentTargetsFilename)
	if err != nil {
		return err
	}
	d.ids = ids
	return nil
}

// isInTargetsMode check if we're passing targets=FILEPATH|Self
func (d *Downloader) isInTargetsMode() bool {
	return d.currentTargetsFilename != ""
}

func (d *Downloader) calculateTotalElements() error {
	fmt.Println("Calculating total elements...")
	if d.isInTargetsMode() {
		// in targets mode, it is important to recalculate TotalElements, since we want
		// the total of each target for a better and cosistant resume
		d.TotalElements = 0
		// make sure we already processed the file before calculated total elements.
		if len(d.ids) == 0 {
			if err := d.processTargetsFile(); err != nil {
				return err
			}
		}
		d.elements = make(map[uint64]uint64, len(d.ids))

		for _, id := range d.ids {
			client.SetTargetPageFilter(id)
			total, err := client.GetTotalElements()
			if err != nil {
				return err
			}
			d.elements[id] = total
			d.TotalElements += total
		}
	} else {
		// unlike targets mode, TotalElements calculation can be skipped.
		// is it already calculated/unmarshaled?
		if d.TotalElements > 0 {
			return nil
		}
		total, err := client.GetTotalElements()
		if err != nil {
			return err
		}
		d.TotalElements = total
		d.CurrentTarget.TotalElements = total
	}

	return nil
}

// Initialize assign parsed flags or params to the local package variables
func Initialize(username string, password string, crawl uint64, mode string,
	noDetails bool, chunkNumber uint64, chunkSize uint64, output string,
	filter string, noResume bool, order string, targets string) error {

	// init client
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

	// does our client setup look good?
	if err := client.IsValid(); err != nil {
		return err
	}

	// init downloader
	downloader = new(output, noResume, targets)

	// can we resume a previous download?
	isResumable, err := downloader.tryResume(noDetails)
	if !isResumable {
		// is it because of an error ? if so, abort
		if err != nil {
			return err
		}

		// no error, start a new download
		fmt.Println("No download to resume; starting a new...")

		// create new outputFile
		newFile, err := os.Create(downloader.OutputFilename)
		if err != nil {
			return err
		}
		outputWriter = bufio.NewWriter(newFile)
	} else {
		// open outputFile
		existingFile, err := os.OpenFile(downloader.OutputFilename, os.O_WRONLY|os.O_APPEND, 0777)
		if err != nil {
			return err
		}
		outputWriter = bufio.NewWriter(existingFile)
	}

	// ensure we have total elemets to download
	downloader.calculateTotalElements()
	fmt.Printf("Total Elements: %d\n", downloader.TotalElements)

	// persist what we have for now for later resumes
	err = downloader.PersistConfig()
	if err != nil {
		return err
	}

	return nil
}

// Get assign params and execute the Run() function
func Get(username string, password string, crawl uint64, mode string,
	deep bool, chunknumber uint64, chunkSize uint64, output string,
	filter string, noResume bool, order string, targets string) error {

	err := Initialize(username, password, crawl, mode, deep, chunknumber, chunkSize,
		output, filter, noResume, order, targets)

	if err != nil {
		return err
	}

	return Run()
}

func (d *Downloader) throttle(timeoutCount *int) {
	*timeoutCount++
	if *timeoutCount >= 3 {
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
			*timeoutCount = 0
		}
	}
}

func (d *Downloader) downloadTarget() error {

	for !d.isDone() {
		var processedLines int64
		debugf("Calling next chunk")
		var chunk []byte
		var statusCode int
		var skip uint64
		err := retry(5, 10, func() error {
			var err error
			chunk, statusCode, skip, err = d.nextChunk()
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
				continue
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
						return fmt.Errorf("\nUnknown error occurred (code %v)", statusCode)
					}
				}
			}
		case statusCode == 504:
			{
				d.throttle(&timeoutCount)
				time.Sleep(time.Second * 30)
				continue
			}
		case statusCode >= 500 && statusCode < 600:
			{
				// meaning: server error
				time.Sleep(time.Second * 30)
				continue
			}
		}

		if statusCode != 200 {
			// just in case it's not an error in the ranges above
			continue
		}

		// iterator for the received chunk
		scanner := bufio.NewScanner(bytes.NewReader(chunk))
		debugf("chunk bytes len: %v", len(chunk))

		// write the header of the tsv only if it's the first/only target
		if d.CurrentTarget.DoneElements == 0 {
			scanner.Scan()
			if d.DoneElements == 0 {
				outputWriter.Write(append(scanner.Bytes(), []byte("\n")...))
			}
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
			d.CurrentTarget.DoneElements++
			d.DoneElements++

			// update the count of lines processed for this chunk
			processedLines++
		}

		// finalize every write
		outputWriter.Flush()

		// save to file the resumer data (to be able to resume later)
		d.PersistConfig()
		debugf("downloader.DoneElements = %v", d.CurrentTarget.DoneElements)

		// scanner error
		if err := scanner.Err(); err != nil {
			errorCount++
			return fmt.Errorf("error wrile scanning chunk: %s", err.Error())
		}

	}
	return nil
}

// Run runs the program
func Run() error {

	// only show progress bar when downloading to file
	if downloader.OutputFilename != "" {
		go progressLoop()
	}

	debug(client.Username, client.Password, client.CrawlID)
	debugf("%#v\n", downloader)

	// var target currentTarget
	var err error

	if downloader.isInTargetsMode() {
		for downloader.TargetsFileNextID < len(downloader.ids) {
			pageID := downloader.ids[downloader.TargetsFileNextID]
			totalElements := downloader.elements[pageID]

			downloader.CurrentTarget.TotalElements = totalElements
			downloader.CurrentTarget.DoneElements = 0

			client.ResetChunkSize()
			client.SetTargetPageFilter(pageID)
			err = downloader.downloadTarget()
			if err != nil {
				return err
			}

			downloader.TargetsFileNextID++
			// downloader.DoneElements += target.DoneElements
			downloader.PersistConfig()
		}
	} else {
		downloader.CurrentTarget = currentTarget{
			TotalElements: downloader.TotalElements,
			DoneElements:  downloader.DoneElements,
		}
		err = downloader.downloadTarget()
		if err != nil {
			// downloader.DoneElements = target.DoneElements
		}
	}

	// when done, remove the resumer file
	if downloader.OutputFilename != "" {
		debugf("removing %v", downloader.getResumeFilename())
		os.Remove(downloader.getResumeFilename())

		// sleep for few millisecond to allow the progress bar to render 100%
		time.Sleep(time.Millisecond * 300)
	}

	return err
}

// nextChunkNumber calculates the index of the next chunk,
// and also returns the number of rows to skip.
// nextChunkNumber is used to calculate the next chunk number after resuming
// and also to recalculate the chunk number in case of throttling.
func (d *Downloader) nextChunkNumber() (nextChunkNumber, skipNRows uint64) {

	// if the remaining elements are less than the page size,
	// request only the remaining elements without having
	// to discard anything.
	remainingElements := d.CurrentTarget.TotalElements - d.CurrentTarget.DoneElements
	if remainingElements < client.ChunkSize && remainingElements > 0 {
		// r.chunkSize = remainingElements
		client.SetChunkSize(remainingElements)
	}

	// if no elements has been downloaded,
	// request the first chunk without skipping rows
	if d.CurrentTarget.DoneElements == 0 {
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

	skipNRows = d.CurrentTarget.DoneElements % client.ChunkSize
	nextChunkNumberFloat, _ := math.Modf(float64(d.CurrentTarget.DoneElements) / float64(client.ChunkSize))

	// just in case nextChunkNumber() gets called when all elements are
	// already downloaded, download chunk and discard all elements
	if d.CurrentTarget.DoneElements == d.CurrentTarget.TotalElements {
		skipNRows = 1
		// r.chunkSize = 1
		client.SetChunkSize(1)
	}

	nextChunkNumber = uint64(nextChunkNumberFloat)
	client.SetNextChunkNumber(nextChunkNumber)
	return
}

// nextChunk configures the API request and returns the chunk
func (d *Downloader) nextChunk() ([]byte, int, uint64, error) {

	_, skipNRows := d.nextChunkNumber()

	if d.CurrentTarget.DoneElements > 0 {
		skipNRows++
	}

	body, statusCode, err := client.FetchRawChunk(false)
	if err != nil {
		return []byte(""), 0, 0, err
	}

	return body, statusCode, skipNRows, nil
}

// PersistConfig saves the resumer to file
func (d *Downloader) PersistConfig() error {
	// save config to file only if not printing to stdout
	if d.OutputFilename == "" {
		return nil
	}

	// if in targets mode, marsha the md5 of the targets file as well, to make sure next time we have a consistent resume
	if d.isInTargetsMode() {
		//  only recalculate the md5 if it's NOT already calculated
		if d.currentTargetsMd5Hash == "" {
			md5, err := getFileMD5Hash(d.currentTargetsFilename)
			if err != nil {
				return err
			}

			d.TargetsFileMD5 = md5
		}

		// update the targets filename to be the currently passed one
		d.TargetsFilename = d.currentTargetsFilename
	}

	config, err := json.MarshalIndent(d, "", "	")
	if err != nil {
		return err
	}

	// create {{output}}.audisto_ file (keeps track of progress etc.)
	return ioutil.WriteFile(d.getResumeFilename(), config, 0644)
}
