package bolt

import (
	"fmt"
	"strings"
	"time"

	"github.com/buger/goterm"
	"github.com/fatih/color"
)

func serverLogger(start time.Time, method string, path string) {
	var colorMethod string

	switch method {
	case "GET":
		colorMethod = color.New(color.FgHiGreen).Sprint(method)
	case "POST":
		colorMethod = color.New(color.FgHiBlue).Sprint(method)
	case "PUT":
		colorMethod = color.New(color.FgHiCyan).Sprint(method)
	case "PATCH":
		colorMethod = color.New(color.FgHiYellow).Sprint(method)
	case "DELETE":
		colorMethod = color.New(color.FgHiRed).Sprint(method)
	default:
		colorMethod = color.New(color.FgHiMagenta).Sprint(method)
	}

	timeString := time.Since(start).String()
	colorTime := color.New(color.FgHiBlack).Sprint(timeString)
	width := goterm.Width()

	dots := strings.Repeat(".", width-len(method)-len(path)-len(timeString)-5 /* 5 spaces */)

	fmt.Printf(" %s %s %s %s \n", colorMethod, path, dots, colorTime)
}
