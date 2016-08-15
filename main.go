package main

import "github.com/audisto/data-downloader/package"

func init() {
	dataDownloader.Initialize()
}
func main() {
	dataDownloader.Run()
}
