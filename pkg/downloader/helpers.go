package downloader

import (
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	// DebugEnvKey debug environment variable
	DebugEnvKey = "DD_DEBUG"
)

// IsInDebugMode checks if the app is running in debug mode
func IsInDebugMode() bool {
	debug := strings.ToLower(os.Getenv(DebugEnvKey))
	for _, activated := range []string{"1", "yes", "y", "true", "t"} {
		if debug == activated {
			return true
		}
	}
	return false
}

// DownloadCompleted a helper function to check if a download for a given output filename has been completed.
// a download is "considered" completed when:
// the output filepath exists + its resume file does not exist
// we're "considering" and not 100% sure since we lack the meta-info resume file.
func DownloadCompleted(outputFilename, resumeFilename string) bool {
	return fExists(resumeFilename) != nil && fExists(outputFilename) == nil
}

func getFileMD5Hash(filepath string) (string, error) {
	infile, inerr := os.Open(filepath)
	if inerr != nil {
		return "", inerr
	}

	md5Hash := md5.New()
	io.Copy(md5Hash, infile)
	return fmt.Sprintf("%x", md5Hash.Sum(nil)), nil
}

// fExists returns nil if path is an existing file/folder
func fExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}
	return nil
}

// random returns a random number in the range between min and max
func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

// chs outputs a string made of c repeated n times
func chs(n int, c string) string {
	var s string
	for i := 0; i < n; i++ {
		s = s + c
	}
	return s
}

// retry operation
func retry(attempts int, sleep int, callback func() error, d *Downloader) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return nil
		}

		if i >= (attempts - 1) {
			break
		}

		errorCount++

		// pause before retrying
		time.Sleep(time.Duration(sleep) * time.Second)

		if d != nil {
			d.debug("Something failed, retrying;")
		}
	}
	return fmt.Errorf("Abandoned after %d attempts, last error: %s", attempts, err)
}
