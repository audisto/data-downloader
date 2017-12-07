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

	// RefreshRate time between to progress updates
	// Export so the caller can fine-tune this
	RefreshRate = time.Millisecond * 100
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
		for {
			select {
			case <-downloader.done:
				close(downloader.status)
				return
			default:
				time.Sleep(RefreshRate)
				downloader.status <- downloader.ProgressReport()
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
