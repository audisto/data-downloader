# Audisto Data-Downloader

[![Build Status](https://travis-ci.org/audisto/data-downloader.svg?branch=master)](https://travis-ci.org/audisto/data-downloader)
 [![GoDoc](https://godoc.org/github.com/audisto/data-downloader?status.svg)](https://godoc.org/github.com/audisto/data-downloader)
 [![Go Report Card](https://goreportcard.com/badge/github.com/audisto/data-downloader)](https://goreportcard.com/report/github.com/audisto/data-downloader)

A command line tool for easy data downloads using the [Audisto](https://audisto.com/) API.

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
  -no-details             If passed, details in API request is set to 0 else to 1
  -no-resume              If passed, download starts again, else the download is resumed
  -filter=[FILTER]        If passed, all pages are filtered by given FILTER
  -order=[ORDER]          If passed, all pages are ordered by given ORDER
```

Start a new download or resume a download with all details:

```shell
$ ./data-downloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --output="myCrawl.tsv"
```

## Installation

You may download compiled executables from the [releases section](https://github.com/audisto/data-downloader/releases).
Download a version for your OS and rename it into ```data-downloader```.

## Installation from Source

Install Go:

Install the Go runtime by downloading the latest release from here: https://golang.org/dl/

Download:

```shell
$ go get -u github.com/audisto/data-downloader
```

Compile:

```shell
$ make build
```
