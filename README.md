# Audisto Data-Downloader

Command line tool for easy data downloads using the Audisto API.

## Installation from Source

Install Go:

Install the Go runtime by downloading the latest release from here: https://golang.org/dl/

Download:

```shell
$ go get -u github.com/audisto/data-downloader
```

Compile:

```shell
$ go build -o data-downloader main.go
```

## Usage

Instructions:

```
usage: data-downloader [options]
	
Parameters:
  -username=[USERNAME]    API Username (required)
  -password=[PASSWORD]    API Password (required)
  -crawl=[ID]             ID of the crawl to download (required)
  -output=[FILE]          Path for the output file
                          If missing the data will be send to the terminal (stdout)
  -no-details             If passed, details in API request is set to 0 else
  -no-resume              If passed, download starts again, else the download is resumed
```

Start a new download or resume a download with all details:

```shell
$ ./data-downloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --output="myCrawl.tsv"
```
