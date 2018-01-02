package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	// File sizes units for formatting
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE

	// ETAFactor ETA milliseconds estimation factor
	ETAFactor = 175
)

// CError a Red-colored Error string (with Ansi escape codes, supports Windows)
func CError(format string, a ...interface{}) error {
	red := color.New(color.FgHiRed).SprintfFunc()
	// Append a new line for a better error readibility.
	return fmt.Errorf(red(format+"\n"), a...)
}

// PrintRed prints a red text into the terminal
func PrintRed(format string, a ...interface{}) {
	color.Red(format, a...)
}

// PrintYellow prints a yellow text into the terminal
func PrintYellow(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// StringYellow format a text with yellow ansi escape codes
func StringYellow(text string) string {
	yellow := color.New(color.FgHiYellow).SprintfFunc()
	return yellow(text)
}

// StringBlue format a text with blue ansi escape codes
func StringBlue(text string) string {
	blue := color.New(color.FgHiBlue).SprintFunc()
	return blue(text)
}

// StringGreen format a text with string ansi escape codes
func StringGreen(text string) string {
	green := color.New(color.FgHiGreen).SprintFunc()
	return green(text)
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

// PrettyTime returns the string representation of the duration. It rounds the time duration to a second and returns a "---" when duration is 0
func PrettyTime(t time.Duration) string {
	if t == 0 {
		return "---"
	}
	return (t - (t % time.Second)).String()
}
