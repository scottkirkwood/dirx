// Dirx utility recursivly gathers folder information by extension
package main

import (
	"flag"
	"fmt"
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
	if err := dirx.Go("/home/scott/20p"); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
