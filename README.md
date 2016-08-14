# data-downloader
Command line tool for easy data downloads using the audisto API.

### Installation

Download:

```shell
$ go get -u github.com/audisto/data-downloader
```

Compile:

```shell
$ go build -o audistoDownloader main.go
```

### Usage

Start new or resume download (all details):

```shell
$ ./audistoDownloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --output="myCrawl.tsv"
```

### Optional parameters

Don't include details: `--no-details`

Don't resume: add `--no-resume`

Output to terminal (stdout): just omit the `--output="file.tsv"` flag
