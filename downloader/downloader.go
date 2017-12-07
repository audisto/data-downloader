package downloader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	// SMOOTHINGFACTOR -
	SMOOTHINGFACTOR = 0.005
	resumerSuffix   = ".audisto_"

	// SelfTargetSuffix used when --targets=self, the output filename will be appended this suffix
	SelfTargetSuffix = "_links"
)

var (
	debugging    = false // if true, debug messages will be shown
	outputWriter *bufio.Writer
)

func init() {
	// check for debug mode once, on package init.
	debugging = IsInDebugMode()
}

// LogType is an alias of string type with predefined log levels.
type LogType string

const (
	INFO    LogType = "INFO"
	WARNING LogType = "WARNING"
	DEBUG   LogType = "DEBUG"
)

// Downloader initiate or resume a persisted downloading process info using AudistoAPIClient
// This also follows and increments chunk number, considering total elements to be downloaded
type Downloader struct {
	OutputFilename            string        `json:"outputFilename"`
	TargetsFilename           string        `json:"targetsFilename"`
	DoneElements              uint64        `json:"doneElements"`
	TotalElements             uint64        `json:"totalElements"`
	NoDetails                 bool          `json:"noDetails"`
	TargetsFileMD5            string        `json:"targetsFileMD5"`
	TargetsFileNextID         int           `json:"targetsFileNextID"`
	CurrentTarget             currentTarget `json:"currentTarget"`
	PagesSelfTargetsCompleted bool          `json:"pagesSelfTargetsCompleted"`

	// Output filename can be change when the downloaded has more than one stage
	// we keep the orginal filename here to be used in suffix/resume operations and checks
	origOutputFilename     string
	noResume               bool
	currentTargetsFilename string
	currentTargetsMd5Hash  string
	ids                    []uint64
	elements               map[uint64]uint64 // [pageID] => totalElements

	// Audisto API client
	client *AudistoAPIClient
	// Report progress via a StatusReport channel
	status chan<- StatusReport
	// make a 'done' channel that tells the progress reporter to stop reporting since we're done
	// declaring 'done' to be of type chan struct{} says that the channel contains no value
	// we’re only interested in its closed property (zero allocation).
	done chan struct{}

	// Store info/warning/debug messages as logs, without printing them these are
	// opt-in, to be communicated through StatusReport channel.
	// No prints should happen mid-execution, prints are the responisibility of caller.
	// errors are not logged, they halt the execution of this downloader and are always returned.
	logs []map[LogType]string
}

// current download target.
// in case of 'targets' mode, this will be dynamic
type currentTarget struct {
	DoneElements  uint64 `json:"doneElements"`
	TotalElements uint64 `json:"totalElements"`
}

// New creates a new downloader
func New(reportProgress chan<- StatusReport) *Downloader {
	if reportProgress != nil {
		return &Downloader{
			status: reportProgress,
			done:   make(chan struct{}),
		}
	}
	return &Downloader{}
}

// getResumeFilename construct the complete file path of the resume file.
// the resume filename is usually the output filename + the resume perfix
// however, --targets=self is a bit tricky and needs a special handling:
// when --targets=self AND the Pages API files is ALREADY downloaded, we auto-switch to:
// output filaname + SelfTargetSuffix + resumerSuffix
func (d *Downloader) getResumeFilename() string {
	defaultPrefixFilename := d.origOutputFilename + resumerSuffix
	if d.isInTargetsMode() {
		// only --targets=self needs special handling
		if d.currentTargetsFilename == "self" {
			// check if Pages API file has FULLY been downloaded
			// FULLY means: the file exists and NO resume file for it
			// or simply PagesSelfTargetsCompleted is marked as true (the case of uninterrupted download)

			// if PagesSelfTargetsCompleted is true no need to check files presence.
			if d.PagesSelfTargetsCompleted {
				return d.getSelfOutputFilename() + resumerSuffix
			}

			// in case of resume, PagesSelfTargetsCompleted might be marked as false but
			// the Pages file IS downloaded.
			if DownloadCompleted(d.origOutputFilename, defaultPrefixFilename) {
				// return the new prefix
				return d.getSelfOutputFilename() + resumerSuffix
			}
		}
	}
	return defaultPrefixFilename
}

func (d *Downloader) getSelfOutputFilename() string {
	// use origOutputFilename instead OutputFilename
	// to make sure we don't get the suffix appended more than once
	org := d.origOutputFilename
	ext := path.Ext(org)
	outfile := org[0:len(org)-len(ext)] + SelfTargetSuffix + ext

	return outfile
}

// tryResume check to see if the current download can be a resume of a previous one
func (d *Downloader) tryResume(noDetails bool) (canBeResumed bool, err error) {

	// Are we outputing to some file?
	if d.OutputFilename == "" || d.noResume {
		return false, nil
	}

	resumeFileExists, outputFileExists := fExists(d.getResumeFilename()), fExists(d.OutputFilename)

	// check if we already have a complete download before?
	if DownloadCompleted(d.OutputFilename, d.getResumeFilename()) {
		if d.currentTargetsFilename == "self" {
			err = fmt.Errorf("%q file and its targets links file seem already downloaded: use --no-resume to create a new", d.OutputFilename)
		} else {
			err = fmt.Errorf("%q file seems already downloaded: use --no-resume to create new", d.OutputFilename)
		}
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
		if d.TargetsFilename == "" && d.currentTargetsFilename != "self" {
			msg := "you are trying to resume a download that had no targets specified before\n"
			msg += "you need to explicitly pass '--no-resume' flag to start a new download"
			return false, fmt.Errorf(msg)
		}

		// In case targets is different than 'self'
		if d.currentTargetsFilename != "self" {
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
		} else { // In case it IS "self" mode
			// check if the download from Pages API stage has been completed.
			if d.PagesSelfTargetsCompleted {
				// if so, make sure we have a consistent resume
				// the previously persisted targetsFileName should be equal to OutputFilename + SelfTargetSuffix
				if d.origOutputFilename != d.TargetsFilename {
					err = fmt.Errorf("resume meta info has been altered, abording an inconsistent resume")
					return false, err
				}

				// update the output filename
				d.OutputFilename = d.getSelfOutputFilename()
				d.currentTargetsFilename = d.TargetsFilename
				// clear filters
				d.client.Filter = ""
				// set mode to links
				d.client.Mode = "links"

				// calculate MD5 hash to ensure the content of the file did not change
				fileMD5, err := getFileMD5Hash(d.currentTargetsFilename)
				if err != nil {
					return false, err
				}

				if fileMD5 != d.TargetsFileMD5 {
					msg := "targets file content has been altered, abording an inconsistent resume.\n"
					msg += "targets filepath: " + d.TargetsFilename + "\n"
					err = fmt.Errorf(msg)
					return false, err
				}
			}
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
	ids, err := d.processTargetFile(d.currentTargetsFilename)
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

// calculateTotalElements calculates the total elements to be downloaded.
func (d *Downloader) calculateTotalElements() error {
	// fmt.Println("Calculating total elements...")
	d.appendLog(INFO, "Calculating total elements...")
	if d.isInTargetsMode() && d.currentTargetsFilename != "self" {
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
			d.client.SetTargetPageFilter(id)
			total, err := d.client.GetTotalElements()
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
		total, err := d.client.GetTotalElements()
		if err != nil {
			return err
		}
		d.TotalElements = total
		d.CurrentTarget.TotalElements = total
	}

	return nil
}

// Setup assign params and execute the Run() function
func (d *Downloader) Setup(username string, password string, crawl uint64, mode string,
	noDetails bool, chunknumber uint64, chunkSize uint64, output string,
	filter string, noResume bool, order string, targets string) error {

	var err error
	// init Audisto client to be used to interact with Audisto Rest API
	d.client, err = NewClient(username, password, crawl, mode, noDetails, chunknumber,
		chunkSize, filter, order)

	if err != nil { // does our client setup look good?
		return err
	}

	// init downloader
	d.OutputFilename = strings.TrimSpace(output)
	d.origOutputFilename = strings.TrimSpace(output)
	d.noResume = noResume
	d.currentTargetsFilename = strings.TrimSpace(targets)

	// can we resume a previous download?
	isResumable, err := d.tryResume(noDetails)

	if !isResumable {
		// is it because of an error ? if so, abort
		if err != nil {
			return err
		}

		// no error, start a new download
		// fmt.Println("No download to resume; starting a new...")
		d.appendLog(INFO, "No download to resume; starting a new...")

		// create new outputFile
		newFile, err := os.Create(d.OutputFilename)
		if err != nil {
			return err
		}
		outputWriter = bufio.NewWriter(newFile)
	} else {
		// open outputFile
		existingFile, err := os.OpenFile(d.OutputFilename, os.O_WRONLY|os.O_APPEND, 0777)
		if err != nil {
			return err
		}
		outputWriter = bufio.NewWriter(existingFile)
	}

	// persist what we have for now for later resumes
	return d.PersistConfig()
}

func (d *Downloader) throttle(timeoutCount *int) {
	*timeoutCount++
	if *timeoutCount >= 3 {
		// throttle
		if (d.client.ChunkSize - 1000) > 0 {

			// if chunkSize is 10000, throttle it down to 7000
			if d.client.ChunkSize == 10000 {
				d.client.ChunkSize -= 3000
			} else {
				// otherwise throttle it down by 1000
				d.client.ChunkSize -= 1000
			}

			// reset the timeout count
			*timeoutCount = 0
		}
	}
}

// downloadTarget use the AudistoAPIClient to download a given target (link or page)
func (d *Downloader) downloadTarget() error {

	for !d.isDone() {
		var processedLines int64
		d.debugf("Calling next chunk")
		var chunk []byte
		var statusCode int
		var chunkStart uint64
		var skip uint64
		err := d.retry(5, 10, func() error {
			var err error
			chunk, statusCode, chunkStart, skip, err = d.nextChunk()
			return err
		})

		if err != nil {
			d.debugf("Too many failures while calling next chunk; %v\n", err)
			return fmt.Errorf("Network error; please check your connection to the internet and resume download")
		}
		d.debugf("Next chunk obtained")
		d.debugf("statusCode: %v", statusCode)

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
		d.debugf("chunk bytes len: %v", len(chunk))

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
			d.debugf("skipping this row: \n%s ", scanner.Text())
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

		scannerErr := scanner.Err()
		if scannerErr == nil {
			// A chunk was completely fetched. Since a chunk may miss lines, adjust resume counter
			d.CurrentTarget.DoneElements = chunkStart + d.client.ChunkSize
		}

		// save to file the resumer data (to be able to resume later)
		d.PersistConfig()
		d.debugf("downloader.DoneElements = %v", d.CurrentTarget.DoneElements)

		// scanner error
		if scannerErr != nil {
			errorCount++
			return fmt.Errorf("Error while scanning chunk: %s", err.Error())
		}

	}
	return nil
}

// Start runs the overall download logic after the initialization and validation steps
func (d *Downloader) Start() error {

	// ensure we have total elemets to download
	d.calculateTotalElements()
	d.appendLog(INFO, fmt.Sprintf("Total Elements: %d", d.TotalElements))

	// Persist calculated total elements to download
	if err := d.PersistConfig(); err != nil {
		return err
	}

	// Report the progress status when the status channel is not nil
	if d.status != nil {
		go reportProgressStatus(d)
	}

	d.debug(d.client.Username, d.client.Password, d.client.CrawlID)
	d.debugf("%#v\n", d)

	// var target currentTarget
	var err error

	// check if we're in targets mode. If so, we'll need to iterate over each target
	// and download it separately. Update `TargetsFileNextID` to keep track of the overall progress.

	// if targets mode is being set to 'self', we need first to download the file containing the links
	// using the pages API then extract link IDs from it, and query the links API for each ID

	// if no targets at all (not in targets mode), we're practically like having one
	// 'target' to download (the pages or the links file).

	if d.isInTargetsMode() {
		if d.currentTargetsFilename != "self" {
			for d.TargetsFileNextID < len(d.ids) {
				pageID := d.ids[d.TargetsFileNextID]
				totalElements := d.elements[pageID]

				d.CurrentTarget.TotalElements = totalElements
				d.CurrentTarget.DoneElements = 0

				d.client.ResetChunkSize()
				d.client.SetTargetPageFilter(pageID)
				err = d.downloadTarget()
				if err != nil {
					return err
				}

				d.TargetsFileNextID++
				// d.DoneElements += target.DoneElements
				d.PersistConfig()
			}
		} else { // self mode, needs a special handling.
			// check if the file containing link IDs has been downloaded using the pages API
			if !d.PagesSelfTargetsCompleted {
				// ensure mode is set to pages
				d.client.Mode = "pages"
				if d.DoneElements > 0 {
					d.appendLog(INFO, "Resuming file download using the Pages API...")
					// fmt.Println("Resuming file download using the Pages API...")
				} else {
					d.appendLog(INFO, "Downloading the file from Pages API...")
					// fmt.Println("Downloading the file from Pages API...")
				}

				d.CurrentTarget = currentTarget{
					TotalElements: d.TotalElements,
					DoneElements:  d.DoneElements,
				}
				err = d.downloadTarget()
				if err != nil {
					return err
				}
				// Once done:
				// - delete this stage's resumer file
				// - Mark the stage as completed,
				// - Switch the targets file from "self" to the newly downloaded filepath
				// - Update the output filepath to the downloaded filepath + SelfTargetSuffix
				// - Persist those in config for resumes, whithin the resumer file of the new filepath (+ SelfTargetSuffix)
				// - switch client mode from Pages to Links
				// - clear filters before using the Links API
				// - reset elements calculation and chunk size
				// - create a a new file and update the output writer

				d.deleteResumerFile()
				// print a informative message about the next stage
				fmt.Println("\nFile downloaded using the Pages API.\nDownloading links...")
				d.PagesSelfTargetsCompleted = true
				d.TargetsFilename = d.origOutputFilename
				d.currentTargetsFilename = d.origOutputFilename
				d.OutputFilename = d.getSelfOutputFilename()
				d.TotalElements = 0
				d.DoneElements = 0
				// reset elements calculation for the new progress report,
				// and since we're going to recalculate the elements for the next stage
				d.CurrentTarget.TotalElements = 0
				d.CurrentTarget.DoneElements = 0
				d.client.ResetChunkSize()
				d.PersistConfig()

				// MAKE SURE filters are cleared once the download using the Pages API is completed
				// before using the Links API.
				d.client.Filter = ""
				// Switch the client mode from pages to links
				d.client.Mode = "links"
				// create the new outputFile
				newFile, err := os.Create(d.OutputFilename)
				if err != nil {
					return err
				}
				outputWriter = bufio.NewWriter(newFile)
				return d.Start() // recursive call to execute the targets stage

			}
		}
	} else {
		d.CurrentTarget = currentTarget{
			TotalElements: d.TotalElements,
			DoneElements:  d.DoneElements,
		}
		err = d.downloadTarget()
		if err != nil {
			return err
		}
	}

	// close the StatusReport channel if it exists
	// do not make this a defer, or move this to the top
	// we have a recursive call of this function when we're in 'targets' mode,
	// and that might close the channel twice.
	if d.status != nil {
		// tell the progress reporter we're done, just by closing it
		close(d.done)
	}

	return d.deleteResumerFile()
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
	if remainingElements < d.client.ChunkSize && remainingElements > 0 {
		// r.chunkSize = remainingElements
		d.client.SetChunkSize(remainingElements)
	}

	// if no elements has been downloaded,
	// request the first chunk without skipping rows
	if d.CurrentTarget.DoneElements == 0 {
		nextChunkNumber = 0
		skipNRows = 0
		d.client.SetNextChunkNumber(0)
		return
	}

	// just in case
	if d.client.ChunkSize < 1 {
		// r.chunkSize = 1
		d.client.SetChunkSize(1)
	}

	skipNRows = d.CurrentTarget.DoneElements % d.client.ChunkSize
	nextChunkNumberFloat, _ := math.Modf(float64(d.CurrentTarget.DoneElements) / float64(d.client.ChunkSize))

	// just in case nextChunkNumber() gets called when all elements are
	// already downloaded, download chunk and discard all elements
	if d.CurrentTarget.DoneElements == d.CurrentTarget.TotalElements {
		skipNRows = 1
		// r.chunkSize = 1
		d.client.SetChunkSize(1)
	}

	nextChunkNumber = uint64(nextChunkNumberFloat)
	d.client.SetNextChunkNumber(nextChunkNumber)
	return
}

// nextChunk configures the API request and returns the chunk
func (d *Downloader) nextChunk() ([]byte, int, uint64, uint64, error) {

	nextChunkNumber, skipNRows := d.nextChunkNumber()
	chunkStartNumber := nextChunkNumber * d.client.ChunkSize

	if d.CurrentTarget.DoneElements > 0 {
		skipNRows++
	}

	if debugging {
		url, _ := d.client.GetRequestURL()
		d.debugf("request url: %s", url.String())
	}

	body, statusCode, err := d.client.FetchRawChunk(false)
	if err != nil {
		return []byte(""), 0, 0, 0, err
	}

	return body, statusCode, chunkStartNumber, skipNRows, nil
}

// PersistConfig saves the resumer to file
func (d *Downloader) PersistConfig() error {
	// save config to file only if not printing to stdout
	if d.OutputFilename == "" {
		return nil
	}

	// if in targets mode, and targets is set to something different than "self",
	// persist the md5 of the targets file as well, to make sure next time we have a consistent resume
	if d.isInTargetsMode() && d.currentTargetsFilename != "self" {
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

func (d *Downloader) deleteResumerFile() error {
	if d.OutputFilename != "" {
		d.debugf("removing %v", d.getResumeFilename())
		return os.Remove(d.getResumeFilename())
	}
	return nil
}

func (d *Downloader) appendLog(logType LogType, message string) {
	log := make(map[LogType]string)
	log[logType] = message
	d.logs = append(d.logs, log)
}

// a shortcut to retry with Downloader receiver
func (d *Downloader) retry(attempts int, sleep int, callback func() error) (err error) {
	return retry(attempts, sleep, callback, d)
}

// processTargetFileLine Process file line according our validation rules:
// If a line:
// - Contains​ only​ digits,​ the​ ID​ is​ the​ line.
// - Starts​ with​ digits​ followed​ by​ a comma,​ the​ ID​ is​ the​ number​ up​ to​ the​ comma.
// - Starts​ with​ digits,​ followed​ by​ whitespace,​ the​ ID​ is​ the​ number​ up​ the​ the whitespace.
// - Does​ not​ start​ with​ a digit,​ it​ is​ ignored.​ A line​ is​ outputted​ stating​ “Line number​ {x}​ was​ ignored”,​ where​ {x}​ is​ the​ number​ of​ the​ current​ line​ (starting​ with​ 1).
// - Does​ start​ with​ digits​ followed​ by​ anything​ but​ whitespace​ or​ a comma,​ it​ is ignored.
func (d *Downloader) processTargetFileLine(line string, lineNumber uint) (valid bool, id uint64) {

	// split the line by whitespaces, tabs if any... using string.Fields
	// this would also respect: if a line contains​ only​ digits, the ID is the line
	// because it's a whole string of digits, well get an array of length 1 and we'll continue processing
	fields := strings.Fields(line)
	if len(fields) < 1 {
		d.appendLog(WARNING, fmt.Sprintf("Line number %d was ignored\n", lineNumber))
		return false, 0
	}

	// remove quoting marks
	relevantString := strings.Trim(fields[0], "\"")
	relevantString = strings.Trim(relevantString, "'")

	// Check the rule: if a line starts​ with​ digits​ followed​ by​ a comma,​ the​ ID​ is​ the​ number​ up​ to​ the​ comma.
	if strings.Contains(relevantString, ",") {
		relevantString = strings.Split(relevantString, ",")[0]
	}

	// Check the rules:
	// - Does​ not​ start​ with​ a digit ..
	// - Does​ start​ with​ digits​ followed​ by​ anything​ but​ whitespace​ or​ a comma
	// Those can be checked at once. by tring to convert the string to a uint64
	// since we already got rid of comma, whitespaces, ..etc
	if value, err := strconv.ParseUint(relevantString, 10, 64); err == nil { // valid line
		return true, value
	}

	d.appendLog(WARNING, fmt.Sprintf("Line number %d was ignored\n", lineNumber))
	return false, 0
}

func (d *Downloader) debugf(format string, a ...interface{}) {
	if debugging {
		d.appendLog(WARNING, fmt.Sprintf(format, a))
	}
}

func (d *Downloader) debug(a ...interface{}) {
	if debugging {
		d.appendLog(WARNING, fmt.Sprint(a))
	}
}

// ProcessTargetFile extract links IDs from a given file
func (d *Downloader) processTargetFile(filePath string) (ids []uint64, err error) {

	file, err := os.Open(filePath)
	defer file.Close()

	if err != nil {
		return ids, err
	}

	scanner := bufio.NewScanner(file)
	var lineNumber uint = 1 // line numbers start with 1 NOT 0

	for scanner.Scan() {
		line := scanner.Text()
		valid, id := d.processTargetFileLine(line, lineNumber)

		lineNumber++ // increment line number first, no matter what
		if !valid {
			continue
		}

		ids = append(ids, id)
	}

	if len(ids) < 1 {
		return ids, fmt.Errorf("targets file does not contain any valid page ID")
	}

	return ids, nil
}
