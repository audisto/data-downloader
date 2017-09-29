package downloader

import (
	"fmt"
	"math/big"
	"time"

	"github.com/gosuri/uilive"
)

// progress bar elements
var (
	progressIndicator *uilive.Writer
	progressStatus    string

	timeoutCount int
	errorCount   int

	averageTimePer1000 float64 = 1
)

// updateStatus sets the first part of the progress bar message
func updateStatus(s string) {
	// TODO: add a mutex?
	progressStatus = s
}

// progress animation
func progressLoop() {

	var n int
	var max = 10

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

// progress outputs the progress percentage
func (r *Resumer) progress() *big.Float {
	var progressPerc *big.Float = big.NewFloat(0.0)
	if res.TotalElements > 0 && res.DoneElements > 0 {
		progressPerc = big.NewFloat(0).Quo(big.NewFloat(100), big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(res.TotalElements), big.NewFloat(0).SetUint64(res.DoneElements)))
	}
	return progressPerc
}
