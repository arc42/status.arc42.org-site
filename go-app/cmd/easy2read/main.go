package main

import (
	"fmt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// format large numbers with separators
func formatWithSeparator(numAsString int) string {

	p := message.NewPrinter(language.German)
	withCommaThousandSep := p.Sprintf("%d", numAsString)
	fmt.Printf("formated string %s", withCommaThousandSep)

	return withCommaThousandSep
}

func main() {
	fmt.Printf("1: %s\n", formatWithSeparator(1))
	fmt.Printf("10: %s\n", formatWithSeparator(10))
	fmt.Printf("100: %s\n", formatWithSeparator(100))
	fmt.Printf("1000: %s\n", formatWithSeparator(1000))
	fmt.Printf("10000: %s\n", formatWithSeparator(10000))
	fmt.Printf("100000: %s\n", formatWithSeparator(100000))
	fmt.Printf("1000000: %s\n", formatWithSeparator(1000000))

}
