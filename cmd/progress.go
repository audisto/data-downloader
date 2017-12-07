package cmd

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/audisto/data-downloader/downloader"

	"github.com/gosuri/uilive"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// RenderProgress render the progressbar animation and the download status information
func RenderProgress(progressReport <-chan downloader.StatusReport) {
	// Make a realtime Stdout writer using uilive
	writer := uilive.New()
	// Make a new progres bar with 100 as its target/percentage
	bar := pb.New(100)
	// By default, the percentage is 0, to be incremented by the progress report
	// percentage := 0
	// Keep track of the time this has been called, so we can know how much time
	// the whole download progress took.
	startTime := time.Now()
	// Don't show the built-in counters
	bar.ShowCounters = false
	// Don't show the built-in time left since we have our own calculation
	bar.ShowTimeLeft = false
	// Don't automatically update the bar, we'll manually update the bar ourselves
	// once we receive an element through the StatusReport channel
	bar.ManualUpdate = true

	// Override the default bar writer, we'd take of the printing using uilive writer
	bar.Output = nil
	// Suppress any prints as well comming from the pb package
	bar.NotPrint = true

	// Custom bar format
	bar.Format("╢▌▌░╟")

	// start the realtime writer
	writer.Start()
	defer writer.Stop() // close the writer at the end of this function no matter what.

	var msg string                           // this hold what will be rendered in Stdout
	var lastProgress downloader.StatusReport // keep a reference to the last status report

	for progress := range progressReport { // for each element received on the channel
		// keep track of the last progress made, we'll need it for stats later
		lastProgress = progress
		// clean the message on each new iteration
		msg = "\n"
		// print the log messages received, BEFORE the progress bar rendering
		for _, f := range progress.Logs {
			for key, value := range f {
				switch key {
				case downloader.INFO:
					msg += StringBlue(value) + "\n"
				case downloader.WARNING:
					msg += StringYellow("WARNING: " + value)
				default:
					msg += StringYellow(value) + "\n"
				}
			}
		}

		// build up the progress bar
		preMsg := fmt.Sprintf("ETA %s |", progress.ETA)
		preMsg += fmt.Sprintf(" Chunk size %d |", progress.ChunkSize)
		preMsg += fmt.Sprintf("%d of %d %s |", progress.DoneElements, progress.TotalElements, progress.Mode)
		preMsg += fmt.Sprintf(" %d Timeouts |", progress.TimeoutsCount)
		preMsg += fmt.Sprintf(" %d Errors ", progress.ErrorsCount)
		bar.Prefix(preMsg)
		percentage := math.Ceil(progress.ProgressPercentage)
		bar.Set(int(percentage))
		bar.Update() // manuall update it

		// the percentage sign '%' of the progress bar is confusing to the writer
		// so we'd escape it before rendering the progress bar.
		bar := "\n" + strings.Replace(bar.String(), "%", "%%", -1)

		msg += bar
		// write all of the above to uilive writer
		fmt.Fprintf(writer, msg+"\n")
	}
	// no more progress is being made
	// reaching this block means the downloader has finished downloading without errors
	// Print some useful basic download stats
	writer.Flush() // flush the previous writer buffer
	finishMessage := "\n\nDownload Completed in " + PrettyTime(time.Since(startTime))

	fi, e := os.Stat(lastProgress.OutputFilename)
	if e == nil {
		filesize := uint64(fi.Size())
		filesizeStr := PrettyByteSize(filesize)
		finishMessage += fmt.Sprintf("\nGot %s Saved to: %s", filesizeStr, lastProgress.OutputFilename)
	}
	msg += StringGreen(finishMessage)
	fmt.Fprintf(writer, msg+"\n")
}
