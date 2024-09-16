package bolt

import (
	"fmt"
	"strings"
	"time"

	"github.com/buger/goterm"
	"github.com/fatih/color"
)

func serverLogger(start time.Time, method string, path string) {
	method, l := methodSpaces(method)
	colorMethod := colorMethodName(method)
	mDots := strings.Repeat(".", l)

	colorMethod = mDots + colorMethod

	timeString := time.Since(start).String()
	colorTime := color.New(color.FgHiBlack).Sprint(timeString)

	width := goterm.Width()
	width = width - len(mDots+method) - len(path) - len(timeString) - 5 /* 5 spaces */

	if width < 5 {
		width = 5
	}

	dots := strings.Repeat(".", width)

	fmt.Printf(" %s %s %s %s \n", colorMethod, path, dots, colorTime)
}
