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

New download (all details):

```shell
$ ./audistoDownloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --output="myCrawl.tsv" --no-resume
```

New download (no details):

```shell
$ ./audistoDownloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --output="myCrawl.tsv" --no-details --no-resume
```

Resume download (all details):

```shell
$ ./audistoDownloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --output="myCrawl.tsv"
```

Resume download (no details):

```shell
$ ./audistoDownloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --output="myCrawl.tsv" --no-details
```


Output to terminal (stdout):

```shell
$ ./audistoDownloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456 --no-details
```

```shell
$ ./audistoDownloader --username="jGSrryHrxtVkxYaONn" --password="UECooHbhYFNBLiIp" --crawl=123456
```