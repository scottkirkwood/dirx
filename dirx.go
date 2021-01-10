package dirx

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"sync"
)

type DirX struct {
	SkipHidden  bool
	FollowLinks bool

	fileChan chan File
	dirChan  chan Dir
	wg       *sync.WaitGroup
}

type Dir struct {
	dirname string
}

type File struct {
	filename string
}

var (
	extRx    = regexp.MustCompile(`.+\.([^.]+)$`)
	hiddenRx = regexp.MustCompile(`^[.][^.]+$`)
)

func NewDirX() *DirX {
	return &DirX{
		fileChan: make(chan File, 1),
		dirChan:  make(chan Dir, 1),
		wg:       &sync.WaitGroup{},
	}
}

func (dx *DirX) Go(path string) error {
	go grabFiles(dx.fileChan)

	dx.wg.Add(1)
	go dx.oneDir(Dir{dirname: path}, dx.dirChan, dx.fileChan, dx.wg)
	go dx.recurseDir(dx.dirChan, dx.fileChan, dx.wg)

	dx.wg.Wait()
	return nil
}

func grabFiles(fileChan chan File) {
	for f := range fileChan {
		fmt.Printf("%q\n", f.filename)
	}
}

func (dx *DirX) recurseDir(dirChan chan Dir, emit chan File, wg *sync.WaitGroup) {
	for dir := range dirChan {
		wg.Add(1)
		go dx.oneDir(dir, dirChan, emit, wg)
	}
}

func (dx *DirX) oneDir(dir Dir, dirChan chan Dir, emit chan File, wg *sync.WaitGroup) {
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
			if dx.addFolder(dirName) {
				dirChan <- Dir{dirname: dirName}
			}
		}
	}
	// Now emit the files
	for _, f := range files {
		name := f.Name()
		if !f.IsDir() && dx.addFile(name) {
			emit <- File{filename: name}
		}
	}
	wg.Done()
}

func (dx *DirX) addFolder(name string) bool {
	if !dx.SkipHidden {
		return true
	}
	return !hiddenRx.MatchString(name)
}

func (dx *DirX) addFile(name string) bool {
	if !dx.SkipHidden {
		return true
	}
	return !hiddenRx.MatchString(name)
}
