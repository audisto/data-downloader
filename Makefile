all: build

dependency:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

test:
	go test $$(go list ./... | grep -v /vendor/)

build:
	go build -o data-downloader main.go

release: release-macosx release-linux release-windows
	
release-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags '-s' -o bin/data-downloader-windows-amd64.exe main.go

release-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags '-s' -o bin/data-downloader-linux-amd64 main.go

release-macosx:
	GOOS=darwin GOARCH=amd64 go build -ldflags '-s' -o bin/data-downloader-macosx-amd64 main.go
