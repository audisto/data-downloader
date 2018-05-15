package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/audisto/data-downloader/pkg/downloader"
	"github.com/gin-gonic/gin"
)

var (
	progressReport chan downloader.StatusReport
	down           *downloader.Downloader
)

func (wd *WebDownloader) homeHandler(c *gin.Context) {
	username, password := getPersistedCredentials()

	c.HTML(http.StatusOK, "home.html", gin.H{
		"username": username,
		"password": password,
	})
}

func (wd *WebDownloader) doLogin(c *gin.Context) {
	var loginPayload Credentials
	err := c.BindJSON(&loginPayload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loginJSON, _ := json.Marshal(loginPayload)
	err = createConfigFile(loginJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

func (wd *WebDownloader) doLogout(c *gin.Context) {
	configPath := getConfigFilePath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"message": "Already logged out"})
		return
	}
	err := os.Remove(configPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (wd *WebDownloader) downloadHandler(c *gin.Context) {
	var downloadOptions JsonPayload

	if wd.downloaderCount == 1 {
		c.JSON(http.StatusConflict, gin.H{"message": "There's already a download in progress"})
		return
	}

	err := c.BindJSON(&downloadOptions)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username, password := downloadOptions.Username, downloadOptions.Password

	if username == "" || password == "" {
		username, password = getPersistedCredentials()
	}

	progressReport = make(chan downloader.StatusReport)
	down = downloader.New(progressReport)

	err = down.Setup(username, password, downloadOptions.CrawlID, downloadOptions.Mode,
		!downloadOptions.Details, 0, 0, downloadOptions.Output, downloadOptions.Filter,
		!downloadOptions.Resume, downloadOptions.Order, "")

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go func() {
		err = down.Start()
		log.Println(err)
		if err != nil {
			message := &ProgressMessage{Error: err.Error()}

			asJSON, err := json.Marshal(message)
			if err != nil {
				fmt.Println(err)
				return
			}
			wd.WebSocket.Broadcast(asJSON)
			return
		}
		wd.downloaderCount++
	}()

	go func() {

		if progressReport != nil {
			for progress := range progressReport {
				if down == nil || down.Stop == true {
					return
				}
				percentage := strconv.FormatFloat(progress.ProgressPercentage, 'f', 2, 64)
				message := &ProgressMessage{
					ProgressPercentage: percentage,
					ETA:                progress.ETA.String(),
					TotalElements:      progress.TotalElements,
					DoneElements:       progress.DoneElements,
					ChunkSize:          progress.ChunkSize,
					ErrorsCount:        progress.ErrorsCount,
				}
				asJSON, err := json.Marshal(message)
				if err != nil {
					fmt.Println(err)
					return
				}
				wd.WebSocket.Broadcast(asJSON)
			}
			wd.downloaderCount--
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "Download started"})
}

func (wd *WebDownloader) stopHandler(c *gin.Context) {
	if down != nil {
		down.Stop = true
		wd.downloaderCount--
		down = nil
		c.JSON(http.StatusOK, gin.H{"message": "Download stopped"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "No download in progress"})
	}
}

func (wd *WebDownloader) progressHandler(c *gin.Context) {
	wd.WebSocket.HandleRequest(c.Writer, c.Request)
}
