package downloader

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

func debugf(format string, a ...interface{}) (n int, err error) {
	if debugging {
		return fmt.Printf("\n"+format+"\n", a...)
	}
	return 0, nil
}

func debug(a ...interface{}) (n int, err error) {
	if debugging {
		return fmt.Println(a...)
	}
	return 0, nil
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

// PrettyTime returns the string representation of the duration. It rounds the time duration to a second and returns a "---" when duration is 0
func PrettyTime(t time.Duration) string {
	if t == 0 {
		return "---"
	}
	return (t - (t % time.Second)).String()
}

// PrettyByteSize returns a human-readable byte string of the form 10M, 12.5K, and so forth.  The following units are available:
//	T: Terabyte
//	G: Gigabyte
//	M: Megabyte
//	K: Kilobyte
//	B: Byte
// The unit that results in the smallest number greater than or equal to 1 is always chosen.
// This is borrowed from: https://github.com/cloudfoundry/bytefmt
func PrettyByteSize(bytes uint64) string {
	unit := ""
	value := float32(bytes)

	switch {
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes >= BYTE:
		unit = "B"
	case bytes == 0:
		return "0"
	}

	stringValue := fmt.Sprintf("%.1f", value)
	stringValue = strings.TrimSuffix(stringValue, ".0")
	return fmt.Sprintf("%s%s", stringValue, unit)
}
