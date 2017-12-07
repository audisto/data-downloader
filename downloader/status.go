package downloader

import (
	"math/big"
	"time"
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

	// RefresInterval time between to progress updates
	// Export so the caller can fine-tune this
	RefresInterval = time.Millisecond * 100
)

// StatusReport a struct holding the progress status of the current download
type StatusReport struct {
	ETA                         time.Duration
	ChunkSize                   uint64
	TotalElements, DoneElements uint64
	Mode                        string
	TimeoutsCount, ErrorsCount  int
	ProgressPercentage          float64
	OutputFilename              string
	Logs                        []map[LogType]string
}

// IsDone a helper function to know if the download is considered done.
func (ps *StatusReport) IsDone() bool {
	return ps.ProgressPercentage >= 100
}

// reportProgressStatus write to the status channel if it exists, the current download progress.
func reportProgressStatus(downloader *Downloader) {
	// don't report progress if we don't have a channel to comminicate with in the first place
	if downloader.status != nil {

		defer downloader.closeStatusChannel()

		for {
			select {
			case <-downloader.done:
				// write for the last time
				downloader.status <- downloader.ProgressReport()
				return
			default:
				downloader.status <- downloader.ProgressReport()
				time.Sleep(RefresInterval)
			}
		}
	}
}

// ProgressReport make the downloader tell its current status
func (d *Downloader) ProgressReport() StatusReport {

	// Calculate Estimated Time of Arival
	ETAuint64, _ := big.NewFloat(0).Quo(big.NewFloat(0).Quo(big.NewFloat(0).Sub(big.NewFloat(0).SetUint64(d.TotalElements), big.NewFloat(0).SetUint64(d.DoneElements)), big.NewFloat(1000)), big.NewFloat(averageTimePer1000)).Uint64()

	// Calculate the progress percentage
	var progressPerc *big.Float = big.NewFloat(0.0)
	var progressF float64
	if d.TotalElements > 0 && d.DoneElements > 0 {
		progressPerc = big.NewFloat(0).Quo(big.NewFloat(100), big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(d.TotalElements), big.NewFloat(0).SetUint64(d.DoneElements)))
		progressF, _ = progressPerc.Float64()
	}

	return StatusReport{
		ETA:                time.Duration(ETAuint64) * time.Millisecond * ETAFactor,
		Mode:               d.client.Mode,
		ChunkSize:          d.client.ChunkSize,
		TotalElements:      d.TotalElements,
		DoneElements:       d.DoneElements,
		TimeoutsCount:      timeoutCount,
		ErrorsCount:        errorCount,
		ProgressPercentage: progressF,
		Logs:               d.logs,
		OutputFilename:     d.OutputFilename,
	}
}

func (d *Downloader) closeStatusChannel() {
	if d.status != nil {
		close(d.status)
		// make the channel nil, so the check d.status != nil becomes meaningful
		// this is important since we might call d.Start() recursively (in targets mode)
		// and close the channel won't mark it as nil, and we'd have a runtime error because
		// we're writing to a closed channel (the 'if' check above will pass).
		d.status = nil
	}
}
