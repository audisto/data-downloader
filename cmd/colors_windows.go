// +build windows

/*
Colors under Windows are tricky. Windows does not support Ansi colors in some
builds (even for Windows 10). Useful links on the matter:
- http://stackoverflow.com/q/44047988/985454
- https://stackoverflow.com/q/16755142
- https://github.com/Microsoft/WSL/issues/1173

While we could workaround this (thanks to go-colorable) for simple prints (like
error messages, usage text, ..etc) we can't easily get this to work with uilive
realtime updates since both have some escape chars.

uilive has its own writer and buffer. And go-colarable also has its own writer
that needs a os.File or a terminal device then checks for Windows on its
own and do the proper escaping. Unfortunaletly there's no "escaped string" under
Windows, go-colorable has to do direct syscalls to support colors under Windows.
uilive might need to be patched for this use case.

*/
package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

// StringYellow returns the plain text as is. Not supported under Windows
func StringYellow(text string) string {
	return text
}

// StringBlue returns the plain text as is. Not supported under Windows
func StringBlue(text string) string {
	return text
}

// StringGreen returns the plain text as is. Not supported under Windows
func StringGreen(text string) string {
	return text
}

// StringRed returns the plain text as is. Not supported under Windows
func StringRed(text string) string {
	return text
}

// CError a Red-colored Error string (with Ansi escape codes, supports Windows)
func CError(format string, a ...interface{}) error {
	// Append a new line for a better error readibility.
	return fmt.Errorf(color.HiRedString(format+"\n", a...))
}

// PrintRed prints a red text into the terminal
func PrintRed(format string, a ...interface{}) {
	color.Red(format, a...)
}

// PrintGreen prints a green text into the terminal
func PrintGreen(format string, a ...interface{}) {
	color.Red(format, a...)
}

// PrintYellow prints a yellow text into the terminal
func PrintYellow(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// PrintBlue prints a blue text into the terminal
func PrintBlue(format string, a ...interface{}) {
	color.Blue(format, a...)
}

// fStringYellow forcibly return a Green string, even under Windwos too.
// should be called when a colored string is needed for "later" simple prints
// (e.g. usage text, but not in uilive updates)
func fStringGreen(text string) string {
	return color.GreenString(text)
}

// fStringYellow forcibly return a Yellow string, even under Windwos too.
// should be called when a colored string is needed for "later" simple prints
// (e.g. usage text, but not in uilive updates)
func fStringYellow(text string) string {
	return color.YellowString(text)
}

// fStringRed forcibly return a Red string, even under Windwos.
// should be called when a colored string is needed for "later" simple prints
// (e.g. usage text, but not in uilive updates)
func fStringRed(text string) string {
	return color.RedString(text)
}
