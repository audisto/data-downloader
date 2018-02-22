package cmd

import (
	"fmt"
	"strings"
	"time"
)

const (
	// File sizes units for formatting
	_byte    = 1.0
	kilobyte = 1024 * _byte
	megabyte = 1024 * kilobyte
	gigabyte = 1024 * megabyte
	terabyte = 1024 * gigabyte

	// ETAFactor ETA milliseconds estimation factor
	ETAFactor = 175
)

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
	case bytes >= terabyte:
		unit = "T"
		value = value / terabyte
	case bytes >= gigabyte:
		unit = "G"
		value = value / gigabyte
	case bytes >= megabyte:
		unit = "M"
		value = value / megabyte
	case bytes >= kilobyte:
		unit = "K"
		value = value / kilobyte
	case bytes >= _byte:
		unit = "B"
	case bytes == 0:
		return "0"
	}

	stringValue := fmt.Sprintf("%.1f", value)
	stringValue = strings.TrimSuffix(stringValue, ".0")
	return fmt.Sprintf("%s%s", stringValue, unit)
}

// PrettyTime returns the string representation of the duration. It rounds the time duration to a second and returns a "---" when duration is 0
func PrettyTime(t time.Duration) string {
	if t == 0 {
		return "---"
	}
	return (t - (t % time.Second)).String()
}
