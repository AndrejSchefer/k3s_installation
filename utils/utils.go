package utils

import (
	"fmt"
	"strings"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

// PrintSectionHeader prints a visual separator, a Colored prefix+message and a blank line.
// msg:    the message to display after the prefix.
// prefix: the prefix tag (e.g. "[INFO]" or "[OK]").
// Color:  the ANSI Color code to apply to the prefix.
// sepLen: how many '-' characters to use for the separator.
func PrintSectionHeader(msg, prefix, Color string, header bool) {
	separator := strings.Repeat("-", 100)

	if header == true {
		fmt.Println("")        // blank line for readability
		fmt.Println("")        // blank line for readability
		fmt.Println(separator) // e.g. "-----…-----"
		// print Colored prefix and message, then reset Color
		fmt.Printf("%s%s%s %s\n", Color, prefix, ColorReset, msg)
		fmt.Println(separator)
	}

	if header == false {
		fmt.Printf("%s%s%s %s\n", Color, prefix, ColorReset, msg)

	}
}
