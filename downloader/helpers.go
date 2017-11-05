package downloader

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DebugEnvKey debug environment variable
	DebugEnvKey = "DD_DEBUG"

	// File sizes units for formatting
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
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

// ProcessTargetFile extract links IDs from a given file
func ProcessTargetFile(filePath string) (ids []uint64, err error) {

	file, err := os.Open(filePath)
	defer file.Close()

	if err != nil {
		return ids, err
	}

	scanner := bufio.NewScanner(file)
	var lineNumber uint = 1 // line numbers start with 1 NOT 0

	for scanner.Scan() {
		line := scanner.Text()
		valid, id := processTargetFileLine(line, lineNumber)

		lineNumber++ // increment line number first, no matter what
		if !valid {
			continue
		}

		ids = append(ids, id)
	}

	if len(ids) < 1 {
		return ids, fmt.Errorf("targets file does not contain any valid page ID")
	}

	return ids, nil
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

// processTargetFileLine Process file line according our validation rules:
// If a line:
// - Contains​ only​ digits,​ the​ ID​ is​ the​ line.
// - Starts​ with​ digits​ followed​ by​ a comma,​ the​ ID​ is​ the​ number​ up​ to​ the​ comma.
// - Starts​ with​ digits,​ followed​ by​ whitespace,​ the​ ID​ is​ the​ number​ up​ the​ the whitespace.
// - Does​ not​ start​ with​ a digit,​ it​ is​ ignored.​ A line​ is​ outputted​ stating​ “Line number​ {x}​ was​ ignored”,​ where​ {x}​ is​ the​ number​ of​ the​ current​ line​ (starting​ with​ 1).
// - Does​ start​ with​ digits​ followed​ by​ anything​ but​ whitespace​ or​ a comma,​ it​ is ignored.
func processTargetFileLine(line string, lineNumber uint) (valid bool, id uint64) {

	// split the line by whitespaces, tabs if any... using string.Fields
	// this would also respect: if a line contains​ only​ digits, the ID is the line
	// because it's a whole string of digits, well get an array of length 1 and we'll continue processing
	fields := strings.Fields(line)
	if len(fields) < 1 {
		fmt.Printf("Line number %d was ignored\n", lineNumber)
		return false, 0
	}

	// remove quoting marks
	relevantString := strings.Trim(fields[0], "\"")
	relevantString = strings.Trim(relevantString, "'")

	// Check the rule: if a line starts​ with​ digits​ followed​ by​ a comma,​ the​ ID​ is​ the​ number​ up​ to​ the​ comma.
	if strings.Contains(relevantString, ",") {
		relevantString = strings.Split(relevantString, ",")[0]
	}

	// Check the rules:
	// - Does​ not​ start​ with​ a digit ..
	// - Does​ start​ with​ digits​ followed​ by​ anything​ but​ whitespace​ or​ a comma
	// Those can be checked at once. by tring to convert the string to a uint64
	// since we already got rid of comma, whitespaces, ..etc
	if value, err := strconv.ParseUint(relevantString, 10, 64); err == nil { // valid line
		return true, value
	}

	fmt.Printf("Line number %d was ignored\n", lineNumber)
	return false, 0
}

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

// retry operation
func retry(attempts int, sleep int, callback func() error) (err error) {
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

		debugf("Something failed, retrying;")
	}
	return fmt.Errorf("Abandoned after %d attempts, last error: %s", attempts, err)
}
