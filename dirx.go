package dirx

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"sort"
	"sync"
	"time"
)

type DirX struct {
	SkipHidden  bool
	FollowLinks bool

	fileChan chan File
	dirChan  chan Dir
	wg       *sync.WaitGroup
	stats    map[string]Stats
}

type Dir struct {
	dirname string
}

type File struct {
	filename string
	size     int64
	time     time.Time
}

type Stats struct {
	ext    string
	count  int
	bytes  int64
	oldest time.Time
	newest time.Time
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
		stats:    make(map[string]Stats),
	}
}

func (dx *DirX) Go(path string) error {
	go dx.gatherFiles(dx.fileChan)

	dx.wg.Add(1)
	go dx.oneDir(Dir{dirname: path}, dx.dirChan, dx.fileChan, dx.wg)
	go dx.recurseDir(dx.dirChan, dx.fileChan, dx.wg)

	dx.wg.Wait()
	return nil
}

func (dx *DirX) gatherFiles(fileChan chan File) {
	for f := range fileChan {
		parts := extRx.FindStringSubmatch(f.filename)
		if len(parts) != 2 {
			continue
		}
		ext := parts[1]
		stats, ok := dx.stats[ext]
		if !ok {
			stats = Stats{
				ext:    ext,
				oldest: f.time,
				newest: f.time,
			}
		}
		stats.count++
		stats.bytes += f.size
		if f.time.After(stats.oldest) {
			stats.oldest = f.time
		}
		if f.time.Before(stats.oldest) {
			stats.newest = f.time
		}
		dx.stats[ext] = stats
	}
}

func (dx *DirX) recurseDir(dirChan chan Dir, emit chan File, wg *sync.WaitGroup) {
	for dir := range dirChan {
		wg.Add(1)
		go dx.oneDir(dir, dirChan, emit, wg)
	}
}

func (dx *DirX) oneDir(dir Dir, dirChan chan Dir, emit chan File, wg *sync.WaitGroup) {
	defer wg.Done()

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
			emit <- File{
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

func (dx *DirX) Print() {
	sorted := dx.toArray()
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})
	for _, stats := range sorted {
		fmt.Printf("%3d *.%s\n", stats.count, stats.ext)
	}
}
