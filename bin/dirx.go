// Dirx utility recursivly gathers folder information by extension
package main

import (
	"flag"
	"fmt"
	"time"
	dirx "github.com/scottkirkwood/dirx"
)

var (
	followFlag     = flag.Bool("l", false, "Follow links")
	skipHiddenFlag = flag.Bool("h", true, "Skip hidden")
)

func main() {
	flag.Parse()

	dirx := dirx.NewDirX()
	dirx.SkipHidden = *skipHiddenFlag
	dirx.FollowLinks = *followFlag
	folder := "."
	if len(flag.Args()) > 0 {
		folder = flag.Arg(0)
	}
	if err := dirx.Go(folder); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	time.Sleep(time.Second)
	dirx.Print()
}
