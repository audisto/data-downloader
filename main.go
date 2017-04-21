package main

import "./package"

func init() {
	dataDownloader.Initialize()
}
func main() {
	dataDownloader.Run()
}
