package dirx

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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
	MaxDepth       int
	Recurse        bool
	SortBySize     bool
	NoCommas       bool

	fileChan      chan File
	dirChan       chan Dir
	gatherFilesWg *sync.WaitGroup
	fileWg        *sync.WaitGroup
	stats         map[string]*Stats
	sorted        []*Stats
	extensionMap  map[string]string
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
		fileChan:      make(chan File),
		dirChan:       make(chan Dir),
		gatherFilesWg: &sync.WaitGroup{},
		fileWg:        &sync.WaitGroup{},
		stats:         make(map[string]*Stats),
	}
}

// Go starts the operation from a certain path
func (dx *DirX) Go(path string) (err error) {
	dx.gatherFilesWg.Add(1)
	go dx.gatherFiles(dx.fileChan)

	go dx.recurseDir()

	dx.fileWg.Add(1)
	dx.oneDir(Dir{dirname: path})

	dx.fileWg.Wait()
	close(dx.fileChan)
	close(dx.dirChan)

	dx.gatherFilesWg.Wait()
	dx.extensionMap, err = dx.makeExtensionMap()
	if err != nil {
		return err
	}
	dx.stats = dx.combineSimilar()
	dx.sorted = dx.toArray()
	return nil
}

func (dx *DirX) toArray() []*Stats {
	ret := make([]*Stats, 0, len(dx.stats))
	for _, stats := range dx.stats {
		ret = append(ret, stats)
	}
	return ret
}

// combine similar, combines extensions like .jpg and .jpeg, .JPG
func (dx *DirX) combineSimilar() map[string]*Stats {
	exts := make(map[string][]string, len(dx.stats))
	for key := range dx.stats {
		toKey := dx.mapExt(key)
		exts[toKey] = append(exts[toKey], key)
	}

	m := make(map[string]*Stats, len(dx.stats))
	for key, stat := range dx.stats {
		toKey := dx.mapExt(key)
		newKey := strings.Join(exts[toKey], ", ")
		m[newKey] = m[newKey].combineStats(stat) // TODO
	}
	return m
}

// fileTypes take from silver search (hg)
var fileTypes = map[string][]string{
	"as":       {"as", "mxml"}, // actionscript
	"ada":      {"ada", "adb", "ads"},
	"asciidoc": {"adoc", "ad", "asc", "asciidoc"},
	"apl":      {"apl"},
	"asm":      {"asm", "s"},
	"asp":      {"asp", "aspx", "asax", "ashx", "ascx", "asmx"},
	"bat":      {"bat", "cmd"},                       // batch
	"bb":       {"bb", "bbappend", "bbclass", "inc"}, // bitbake
	"c":        {"c", "h", "xs"},
	"cfc":      {"cfc", "cfm", "cfml"},                 // cfmx
	"clj":      {"clj", "cljs", "cljc", "cljx", "edn"}, // clojure
	"coffee":   {"coffee", "cjsx"},
	"coq":      {"coq", "g"},
	"cpp":      {"cpp", "cc", "C", "cxx", "m", "hpp", "hh", "H", "hxx", "tpp"},
	"cr":       {"cr", "ecr"},                                                                                  // crystal
	"pyx":      {"pyx", "pxd", "pxi"},                                                                          // cython
	"pas":      {"pas", "int", "dfm", "nfm", "dof", "dpk", "dpr", "dproj", "groupproj", "bdsgroup", "bdsproj"}, // delphi
	"d":        {"d", "di"},                                                                                    // dlang
	"dot":      {"dot", "gv"},
	"dts":      {"dts", "dtsi"},
	"ebuild":   {"ebuild", "eclass"},
	"ex":       {"ex", "eex", "exs"},                                                      // elixir
	"erl":      {"erl", "hrl"},                                                            // erlang
	"f":        {"f", "F", "f77", "f90", "F90", "f95", "f03", "for", "ftn", "fpp", "FPP"}, // fortran
	"fs":       {"fs", "fsi", "fsx"},                                                      // fsharp
	"po":       {"po", "pot", "mo"},                                                       // gettext
	"vert":     {"vert", "tesc", "tese", "geom", "frag", "comp"},                          // glsl
	"groovy":   {"groovy", "gtmpl", "gpp", "grunit", "gradle"},
	"hs":       {"hs", "hsig", "lhs"}, // haskell
	"html":     {"htm", "html", "shtml", "xhtml"},
	"idr":      {"idr", "ipkg", "lidr"}, // idris
	"java":     {"java", "properties"},
	"js":       {"es6", "js", "jsx", "vue"},
	"json":     {"json"},
	"jsp":      {"jsp", "jspx", "jhtm", "jhtml", "jspf", "tag", "tagf"},
	"lisp":     {"lisp", "lsp"},
	"mak":      {"Makefiles", "Makefile", "mk", "mak"},
	"markdown": {"markdown", "mdown", "mdwn", "mkdn", "mkd", "md"},
	"mas":      {"mas", "mhtml", "mpl", "mtxt"},           // mason
	"asa":      {"asa", "rsa"},                            // naccess
	"ml":       {"ml", "mli", "mll", "mly"},               // ocaml
	"pir":      {"pir", "pasm", "pmc", "ops", "pg", "tg"}, // parrot
	"pl":       {"pl", "pm", "pm6", "t"},                  // perl
	"php":      {"php", "phpt", "php3", "php4", "php5", "phtml"},
	"pike":     {"pike", "pmod"},
	"pt":       {"pt", "cpt", "metadata", "cpy", "zcml"}, // plone
	"r":        {"r", "R", "Rmd", "Rnw", "Rtex", "Rrst"},
	"rb":       {"rb", "rhtml", "rjs", "rxml", "erb", "rake", "spec"}, // ruby
	"sass":     {"sass", "scss"},
	"sh":       {"sh", "bash", "csh", "tcsh", "ksh", "zsh", "fish"}, // shell
	"sml":      {"sml", "fun", "mlb", "sig"},
	"ado":      {"do", "ado"}, // stata
	"tcl":      {"tcl", "itcl", "itk"},
	"tf":       {"tf", "tfvars"}, // terraform
	"tex":      {"tex", "sty"},
	"tt":       {"tt", "tt2", "ttml"},
	"ts":       {"ts", "tsx"},
	"vala":     {"vala", "vapi"},
	"vb":       {"bas", "frm", "vb", "resx"},
	"vm":       {"vm", "vtl", "vsl"},     // velocity
	"v":        {"v", "vh", "sv", "svh"}, // verilog
	"vhdl":     {"vhd", "vhdl"},
	"wxi":      {"wxi", "wxs"}, // wix
	"xml":      {"xml", "dtd", "xsl", "xslt", "xsd", "ent", "tld", "plist", "wsdl"},
	"yaml":     {"yaml", "yml"},
	"zeek":     {"zeek", "bro", "bif"},
	// Images (wiki)
	"jpeg": {"jpg", "jpeg", "jpe", "jif", "jfif", "jfi"},
	"tiff": {"tiff", "tif"},
	"raw":  {"raw", "arw", "cr2", "nrw", "k25"},
	"bmp":  {"bmp", "dib"},
	"heif": {"heif", "heic"},
	"indd": {"ind", "indd", "indt"},
	"jp2":  {"jp2", "j2k", "jpf", "jpx", "jpm", "mj2"},
	"svg":  {"svg", "svgz"},
	// Video (Internet)
	"ogg":  {"ogv", "ogg"},
	"mts":  {"mts", "m2ts"},
	"mov":  {"mov", "qt"},
	"mp4":  {"mp4", "m4p", "m4v"},
	"mpeg": {"mpg", "mp2", "mpeg", "mpe", "mpv"},
	"flv":  {"flv", "f4v", "f4p", "f4a", "f4b"}, // flash
}

// makeExtensionMap makes a map of multiple extensions to one extension
func (dx *DirX) makeExtensionMap() (map[string]string, error) {
	m := map[string]string{}
	for k, types := range fileTypes {
		found := false
		for _, v := range types {
			if v == k {
				found = true
				break
			}
		}
		if !found {
			return m, fmt.Errorf("did not find key in values %q {%v}", k, types)
		}
		for _, v := range types {
			x, ok := m[v]
			if ok {
				return m, fmt.Errorf("found duplicate value %q to key %q at %q: {%v}", v, x, k, types)
			}
			m[v] = k
		}
	}
	return m, nil
}

func (dx *DirX) mapExt(ext string) string {
	to, ok := dx.extensionMap[ext]
	if ok {
		return to
	}
	ext = strings.ToLower(ext)
	to, ok = dx.extensionMap[ext]
	if ok {
		return to
	}
	return ext
}

// Sort sorts the values with the given sort criteria.
func (dx *DirX) Sort() {
	sort.Slice(dx.sorted, func(i, j int) bool {
		return dx.Less(dx.sorted[i], dx.sorted[j])
	})
}

// Less returns true if a is less than b
func (dx *DirX) Less(a, b *Stats) bool {
	if dx.SortBySize {
		if a.bytes == b.bytes {
			return strings.ToLower(a.ext) < strings.ToLower(b.ext)
		}
		return a.bytes > b.bytes
	}
	if a.count == b.count {
		return strings.ToLower(a.ext) < strings.ToLower(b.ext)
	}
	return a.count > b.count
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
			stats = &Stats{
				ext:      ext,
				oldest:   f.time,
				newest:   f.time,
				smallest: 2e6,
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
		if f.time.Before(stats.newest) {
			stats.newest = f.time
		}
		dx.stats[ext] = stats
	}
}

func (s *Stats) combineStats(stat *Stats) *Stats {
	if s == nil {
		return &Stats{
			ext:       stat.ext,
			firstFile: stat.firstFile,
			count:     stat.count,
			bytes:     stat.bytes,
			smallest:  stat.smallest,
			largest:   stat.largest,
			oldest:    stat.oldest,
			newest:    stat.newest,
		}
	}
	s.count += stat.count
	s.bytes += stat.bytes
	s.ext = s.ext + ", " + stat.ext
	if s.smallest < stat.smallest {
		s.smallest = stat.smallest
	}
	if s.largest > stat.largest {
		s.largest = stat.largest
	}
	if s.oldest.After(stat.oldest) {
		s.oldest = stat.oldest
	}
	if s.newest.Before(stat.newest) {
		s.newest = stat.newest
	}
	return s
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

	files, err := os.ReadDir(dir.dirname)
	if err != nil {
		if !strings.Contains(err.Error(), "permission denied") {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}
	// Emit the folders
	for _, f := range files {
		d := Dir{dirname: path.Join(dir.dirname, f.Name())}
		if f.IsDir() && dx.addFolder(d) {
			dx.fileWg.Add(1)
			dx.dirChan <- d
		}
	}
	// Emit the files
	for _, f := range files {
		name := f.Name()
		if !f.IsDir() && dx.addFile(name) {
			i, _ := f.Info()
			dx.fileChan <- File{
				filename: name,
				size:     i.Size(),
				time:     i.ModTime(),
			}
		}
	}
}

func (dx *DirX) depthIsOk(dir Dir) bool {
	if dx.Recurse && dx.MaxDepth <= 0 {
		return true
	}
	depth := dir.depth()
	if !dx.Recurse && depth > 0 {
		return false
	}
	return dir.depth() < dx.MaxDepth
}

func (dx *DirX) filenameIsOk(fname string) bool {
	if !dx.SkipHidden {
		return true
	}
	return !hiddenRx.MatchString(fname)
}

func (dx *DirX) addFolder(dir Dir) bool {
	if !dx.depthIsOk(dir) {
		return false
	}
	if !dx.filenameIsOk(dir.baseName()) {
		return false
	}
	return true
}

func (dx *DirX) addFile(name string) bool {
	return dx.filenameIsOk(name)
}

func (d Dir) depth() int {
	if d.dirname == "." {
		return 0
	}
	return 1 + strings.Count(d.dirname, "/")
}

func (d Dir) baseName() string {
	return filepath.Base(d.dirname)
}
