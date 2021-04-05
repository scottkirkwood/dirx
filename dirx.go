package dirx

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// DirX is the main structure to perform DirX operations
type DirX struct {
	SkipHidden     bool
	FollowLinks    bool
	ShowSingleName bool

	fileChan      chan File
	dirChan       chan Dir
	gatherFilesWg *sync.WaitGroup
	fileWg        *sync.WaitGroup
	stats         map[string]Stats
	sorted        []Stats
}

// Dir is just the relative directory name
type Dir struct {
	dirname string
}

// File is the filename plus some other bits of data
type File struct {
	filename string
	size     int64
	time     time.Time
}

// Stats keeps track of extension statistics
type Stats struct {
	ext       string
	firstFile string
	count     int
	bytes     int64
	smallest  int64
	largest   int64
	oldest    time.Time
	newest    time.Time
}

var (
	extRx    = regexp.MustCompile(`.+\.([^.]+)$`)
	hiddenRx = regexp.MustCompile(`^[.][^.]+$`)
)

// NewDirX creates a new empty DirX object
func NewDirX() *DirX {
	return &DirX{
		fileChan:      make(chan File, 0),
		dirChan:       make(chan Dir, 0),
		gatherFilesWg: &sync.WaitGroup{},
		fileWg:        &sync.WaitGroup{},
		stats:         make(map[string]Stats),
	}
}

// Go starts the operation from a certain path
func (dx *DirX) Go(path string) error {
	dx.gatherFilesWg.Add(1)
	go dx.gatherFiles(dx.fileChan)

	go dx.recurseDir()

	dx.fileWg.Add(1)
	dx.oneDir(Dir{dirname: path})

	dx.fileWg.Wait()
	close(dx.fileChan)
	close(dx.dirChan)

	dx.gatherFilesWg.Wait()
	dx.sorted = dx.toArray()
	return nil
}

// Sort sorts the values with the given sort criteria.
func (dx *DirX) Sort() {
	sort.Slice(dx.sorted, func(i, j int) bool {
		if dx.sorted[i].count == dx.sorted[j].count {
			return strings.ToLower(dx.sorted[i].ext) < strings.ToLower(dx.sorted[j].ext)
		}
		return dx.sorted[i].count > dx.sorted[j].count
	})
}

// gatherFiles needs to run in a goroutine and gathers statistics
// over files in the fileChan
func (dx *DirX) gatherFiles(fileChan chan File) {
	defer dx.gatherFilesWg.Done()

	for f := range fileChan {
		parts := extRx.FindStringSubmatch(f.filename)
		if len(parts) != 2 {
			continue
		}
		ext := parts[1]
		stats, ok := dx.stats[ext]
		if !ok {
			stats = Stats{
				ext:      ext,
				oldest:   f.time,
				newest:   f.time,
				smallest: 2E6,
			}
		}
		stats.count++
		if stats.count == 1 {
			stats.firstFile = f.filename
		}
		stats.bytes += f.size
		if f.size < stats.smallest {
			stats.smallest = f.size
		}
		if f.size > stats.largest {
			stats.largest = f.size
		}
		if f.time.After(stats.oldest) {
			stats.oldest = f.time
		}
		if f.time.Before(stats.oldest) {
			stats.newest = f.time
		}
		dx.stats[ext] = stats
	}
}

// recurseDir performs a breadth first search over the folders by using
// the dirChan and should run in a goroutine
func (dx *DirX) recurseDir() {
	for dir := range dx.dirChan {
		go dx.oneDir(dir)
	}
}

// oneDir emits File and Dir channels as it iterates over one directory
func (dx *DirX) oneDir(dir Dir) {
	defer dx.fileWg.Done()

	files, err := ioutil.ReadDir(dir.dirname)
	if err != nil {
		if !strings.Contains(err.Error(), "permission denied") {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}
	// Emit the folders
	for _, f := range files {
		name := f.Name()
		if f.IsDir() && dx.addFolder(name) {
			dirName := path.Join(dir.dirname, name)
			dx.fileWg.Add(1)
			dx.dirChan <- Dir{dirname: dirName}
		}
	}
	// Emit the files
	for _, f := range files {
		name := f.Name()
		if !f.IsDir() && dx.addFile(name) {
			dx.fileChan <- File{
				filename: name,
				size:     f.Size(),
				time:     f.ModTime(),
			}
		}
	}
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

func (dx *DirX) toArray() []Stats {
	ret := make([]Stats, 0, len(dx.stats))
	for _, stats := range dx.stats {
		ret = append(ret, stats)
	}
	return ret
}
