all: build

dependency:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

embed-static:
	@echo "embedding static files"
	go generate

test: embed-static
	go test $$(go list ./... | grep -v /vendor/)

build: embed-static
	go build -o data-downloader main.go

release: embed-static release-macosx release-linux release-windows
	
release-windows: embed-static
	GOOS=windows GOARCH=amd64 go build -ldflags '-s -w' -o bin/data-downloader-windows-amd64.exe main.go

release-linux: embed-static
	GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o bin/data-downloader-linux-amd64 main.go

release-macosx: embed-static
	GOOS=darwin GOARCH=amd64 go build -ldflags '-s -w' -o bin/data-downloader-macosx-amd64 main.go
