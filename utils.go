package bolt

import (
	"fmt"
	"strings"

	"github.com/buger/goterm"
	"github.com/fatih/color"
)

func displayServeInfo(listenAddr string, mode Mode) {
	c := color.New(color.FgMagenta)
	c.Println("    ____        ____ ")
	c.Println("   / __ )____  / / /_")
	c.Println("  / __  / __ \\/ / __/")
	c.Println(" / /_/ / /_/ / / /_", color.New(color.FgHiYellow).Sprint(mode))
	c.Println("/_____/\\____/_/\\__/", color.New(color.FgHiGreen).Sprintf("v%s", Version))

	c.Printf("↳ Server listening on %s\n\n", listenAddr)

	if mode == Development {
		c = color.New(color.FgRed, color.Bold)
		c.Println("Running in development mode. Do not use in production! 🚨")
	}
}

func methodSpaces(method string) (string, int) {
	var (
		max = 6
		l   = max - len(method)
	)
	if len(method) > max {
		l = 0
	}
	return method, l
}

func colorMethodName(method string) string {
	switch strings.TrimSpace(method) {
	case "GET":
		return color.New(color.FgHiGreen).Sprint(method)
	case "POST":
		return color.New(color.FgHiBlue).Sprint(method)
	case "PUT":
		return color.New(color.FgHiCyan).Sprint(method)
	case "PATCH":
		return color.New(color.FgHiYellow).Sprint(method)
	case "DELETE":
		return color.New(color.FgHiRed).Sprint(method)
	default:
		return color.New(color.FgHiMagenta).Sprint(method)
	}
}

func logRoutes(routes []Route) {
	for _, route := range routes {
		for _, p := range route.NormalizedPaths() {
			method, l := methodSpaces(route.Method())
			colorMethod := colorMethodName(method)
			mDots := strings.Repeat(".", l)

			colorMethod = mDots + colorMethod

			name := route.GetName()
			if name == "" {
				name = "unnamed"
			}

			width := goterm.Width()
			width = width - len(mDots+method) - len(p) - len(name) - 5 /* 5 spaces */

			if width < 5 {
				width = 5
			}

			dots := strings.Repeat(".", width)

			fmt.Printf(" %s %s %s %s \n", colorMethod, p, dots, color.New(color.FgHiBlack).Sprint(name))
		}
	}
}
