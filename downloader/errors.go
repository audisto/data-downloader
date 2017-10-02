package downloader

// StatusCodesErrors ..
var StatusCodesErrors = map[int]string{
	401: "Wrong credentials",
	403: "Access denied. Wrong credentials?",
	404: "Not found. Correct crawl ID?",
	429: "Error while getting total number of elements: 429, multiple requests",
	504: "Error while getting total number of elements: 504, server timeout",
}
