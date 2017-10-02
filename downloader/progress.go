package downloader

import (
	"fmt"
	"math/big"
	"os"
	"time"

	pb "gopkg.in/cheggaaa/pb.v1"
)

const (
	// ETAFactor ETA milliseconds estimation factor
	ETAFactor = 175
)

// progress bar elements
var (
	timeoutCount int
	errorCount   int

	averageTimePer1000 float64 = 1
)

// progress animation
func progressLoop() {
	fmt.Println() // print a new line before showing the progress bar
	percentage := 0
	bar := pb.StartNew(100)
	bar.ShowCounters = false
	bar.ShowTimeLeft = false
	startTime := time.Now()
	bar.Format("╢▌▌░╟")
	for percentage < 100 {
		if (downloader.TotalElements > 0) {
			percentage = int((downloader.DoneElements * 100) / downloader.TotalElements)
		}

		ETAuint64, _ := big.NewFloat(0).Quo(big.NewFloat(0).Quo(big.NewFloat(0).Sub(big.NewFloat(0).SetUint64(downloader.TotalElements), big.NewFloat(0).SetUint64(downloader.DoneElements)), big.NewFloat(1000)), big.NewFloat(averageTimePer1000)).Uint64()
		ETAtime := time.Duration(ETAuint64) * time.Millisecond * ETAFactor
		ETAstring := PrettyTime(ETAtime)

		preMsg := fmt.Sprintf("ETA %s |", ETAstring)
		preMsg += fmt.Sprintf(" Chunk size %d |", client.ChunkSize)
		preMsg += fmt.Sprintf("%d of %d %s |", downloader.DoneElements, downloader.TotalElements, client.Mode)
		preMsg += fmt.Sprintf(" %d Timeouts |", timeoutCount)
		preMsg += fmt.Sprintf(" %d Errors ", errorCount)
		bar.Prefix(preMsg)
		bar.Set(percentage)
		time.Sleep(time.Millisecond * 100)

	}

	bar.FinishPrint(fmt.Sprintf("\nDwonload Completed in %s", PrettyTime(time.Since(startTime))))
	fi, e := os.Stat(downloader.OutputFilename)
	if e == nil {
		filesize := uint64(fi.Size())
		bar.FinishPrint(fmt.Sprintf("Got %s Saved to: %s", PrettyByteSize(filesize), downloader.OutputFilename))
	}
}

// progress outputs the progress percentage
func (r *Downloader) progress() *big.Float {
	var progressPerc *big.Float = big.NewFloat(0.0)
	if downloader.TotalElements > 0 && downloader.DoneElements > 0 {
		progressPerc = big.NewFloat(0).Quo(big.NewFloat(100), big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(downloader.TotalElements), big.NewFloat(0).SetUint64(downloader.DoneElements)))
	}
	return progressPerc
}
