package main

import "github.com/audisto/data-downloader/downloader"

func init() {
	downloader.Initialize()
}

func main() {
	downloader.Run()
}
