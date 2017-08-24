package main

import (
	"fmt"
)

func moveCursor(x, y int) {
	fmt.Printf("\x1b[%d;%dH", y+1, x+1)
}

func printf(
	colorizer func(string) string,
	format string,
	args ...interface{},
) {
	fmt.Print(colorizer(fmt.Sprintf(format, args...)))
}
