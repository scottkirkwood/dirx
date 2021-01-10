// Dirx utility recursivly gathers folder information by extension
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"sync"
	"time"
)

var (
	extRx    = regexp.MustCompile(`.+\.([^.]+)$`)
	hiddenRx = regexp.MustCompile(`^[.][^.]+$`)
)

var (
	followFlag     = flag.Bool("l", false, "Follow links")
	skipHiddenFlag = flag.Bool("h", true, "Skip hidden")
)

type Dir struct {
	dirname string
}

type File struct {
	filename string
}

func grabFiles(fileChan chan File) {
	for f := range fileChan {
		fmt.Printf("%q\n", f.filename)
	}
}

func recurseDir(dirChan chan Dir, emit chan File, wg *sync.WaitGroup) {
	for dir := range dirChan {
		wg.Add(1)
		go oneDir(dir, dirChan, emit, wg)
	}
}

func oneDir(dir Dir, dirChan chan Dir, emit chan File, wg *sync.WaitGroup) {
	fmt.Printf("dir %q\n", dir.dirname)
	files, err := ioutil.ReadDir(dir.dirname)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	// Add all folders to dirChan
	for _, f := range files {
		name := f.Name()
		if f.IsDir() {
			dirName := path.Join(dir.dirname, name)
			if addFolder(dirName) {
				dirChan <- Dir{dirname: dirName}
			}
		}
	}
	// Now emit the files
	for _, f := range files {
		name := f.Name()
		if !f.IsDir() && addFile(name) {
			emit <- File{filename: name}
		}
	}
	wg.Done()
}

func addFolder(name string) bool {
	if !*skipHiddenFlag {
		return true
	}
	return !hiddenRx.MatchString(name)
}

func addFile(name string) bool {
	if !*skipHiddenFlag {
		return true
	}
	return !hiddenRx.MatchString(name)
}

func main() {
	flag.Parse()

	fileChan := make(chan File, 1)
	dirChan := make(chan Dir, 1)
	wg := &sync.WaitGroup{}

	go grabFiles(fileChan)

	wg.Add(1)
	go oneDir(Dir{dirname: "/home/scott/20p"}, dirChan, fileChan, wg)
	go recurseDir(dirChan, fileChan, wg)

	wg.Wait()
	time.Sleep(1 * time.Second)
}
