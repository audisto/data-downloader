#!/usr/bin/env bash
GOOS=linux GOARCH=amd64  go build -o bin/data-downloader main.go
GOOS=windows GOARCH=amd64  go build -o bin/data-downloader.exe main.go
GOOS=darwin GOARCH=amd64  go build -o bin/data-downloader.run main.go
