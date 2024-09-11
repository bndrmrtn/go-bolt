package bolt

import (
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
}
