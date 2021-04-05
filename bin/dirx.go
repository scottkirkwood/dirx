// Dirx utility recursivly gathers folder information by extension
package main

import (
	"flag"
	"fmt"
	dirx "github.com/scottkirkwood/dirx"
	"os"
	"os/signal"
	"syscall"
)

var (
	followFlag     = flag.Bool("l", false, "Follow links")
	skipHiddenFlag = flag.Bool("h", true, "Skip hidden")
	showSingleNameFlag = flag.Bool("only", false, "Show full name if it's the only one")
)

func main() {
	flag.Parse()

	// Swallow the "signal: broken pipe" message
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGPIPE)

	dirx := dirx.NewDirX()
	dirx.SkipHidden = *skipHiddenFlag
	dirx.FollowLinks = *followFlag
	dirx.ShowSingleName = *showSingleNameFlag
	folder := "."
	if len(flag.Args()) > 0 {
		folder = flag.Arg(0)
	}

	if err := dirx.Go(folder); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	dirx.Sort()
	dirx.Print()
}
