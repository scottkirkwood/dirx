// Dirx utility recursivly gathers folder information by extension
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	dirx "github.com/scottkirkwood/dirx"
)

var (
	followFlag         = flag.Bool("l", false, "Follow links")
	skipHiddenFlag     = flag.Bool("h", true, "Skip hidden")
	recurseFlag        = flag.Bool("r", false, "Recurse into subdirectories (equivalent to -maxdepth=1)")
	bySizeFlag         = flag.Bool("s", false, "Sort by size (defaults to extension)")
	maxDepthFlag       = flag.Int("maxdepth", 0, "Maximum depth 0 means infinite, 1 is current directory")
	showSingleNameFlag = flag.Bool("only", false, "Show full name if it's the only one")
	noCommasFlag       = flag.Bool("numbers", false, "Numbers with no thousands separator")
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
	dirx.Recurse = *recurseFlag
	dirx.MaxDepth = *maxDepthFlag
	dirx.SortBySize = *bySizeFlag
	dirx.NoCommas = *noCommasFlag
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
