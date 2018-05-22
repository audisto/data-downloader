package web

import (
	"github.com/audisto/data-downloader/pkg/downloader"
	"gopkg.in/olahol/melody.v1"
)

const (
	audistoHomeDirecotyName    = ".audisto"
	audistoCredentialsFileName = "credentials.json"
)

// Credentials holds the json payload and for the login username and password
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type JsonPayload struct {
	CrawlID  uint64 `json:"crawlID,string"`
	Mode     string `json:"mode"`
	Filter   string `json:"filter"`
	Order    string `json:"order"`
	Resume   bool   `json:"resume"`
	Details  bool   `json:"details"`
	Target   string `json:"target"`
	Output   string `json:"output"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ProgressMessage struct {
	ETA                  string `json:"ETA"`
	ChunkSize            uint64 `json:"chunkSize,string"`
	TotalElements        uint64 `json:"totalElements,string"`
	DoneElements         uint64 `json:"doneElements,string"`
	Mode                 string `json:"mode"`
	TimeoutsCount        int    `json:"timoutsCount,string"`
	ErrorsCount          int    `json:"errorsCount,string"`
	ProgressPercentage   string `json:"progressPercentage"`
	OutputFilename       string `json:"outputFilename"`
	LogMessage           string `json:"logMessage"`
	IsIngTargetMode      bool   `json:"isTargetMode"`
	TotalIDsCount        int    `json:"totalIDsCount"`
	CurrentIDOrderNumber int    `json:"currentIDOrderNumber"`
	Error                string `json:"error"`
}

type WebDownloader struct {
	IsDone          bool
	IsStopped       bool
	APIDownloader   *downloader.Downloader
	JSONPayload     JsonPayload
	WebSocket       *melody.Melody
	downloaderCount int // Restrict the number of parallel downloads
}

// NewWebDownloader -
func NewWebDownloader() *WebDownloader {
	return &WebDownloader{
		WebSocket: melody.New(),
	}
}
