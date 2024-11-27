package dirx

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/text/message"
)

var printer = message.NewPrinter(message.MatchLanguage("en"))

type displayRows struct {
	rows       [][]string
	colWidths  []int
	rightAlign []bool
}

func (r *displayRows) addRow(cols ...string) {
	if len(r.colWidths) == 0 {
		r.colWidths = make([]int, len(cols))
	}
	row := make([]string, len(cols))
	r.rows = append(r.rows, row)
	for i, col := range cols {
		row[i] = col
		if len(col) > r.colWidths[i] {
			r.colWidths[i] = len(col)
		}
	}
}

func (r *displayRows) getRow(index int) string {
	cols := make([]string, len(r.rows[index]))
	for i, width := range r.colWidths {
		if r.rightAlign[i] {
			cols[i] = fmt.Sprintf("%*s", width, r.rows[index][i])
		} else {
			cols[i] = fmt.Sprintf("%-*s", width, r.rows[index][i])
		}
	}
	return strings.Join(cols, " ")
}

// Len is the standard length interface
func (r *displayRows) Len() int {
	return len(r.rows)
}

// Len is the standard length interface
func (r *displayRows) RightAlign(rightAligns ...bool) {
	r.rightAlign = make([]bool, len(rightAligns))
	for i, a := range rightAligns {
		r.rightAlign[i] = a
	}
}

// Print prints out the values
func (dx *DirX) Print() {
	width, _ := getColWidth()
	_ = width // TODO use the terminal width
	rows := dx.formatRows()
	for i := 0; i < rows.Len(); i++ {
		fmt.Println(rows.getRow(i))
	}
}

func (dx *DirX) formatRows() displayRows {
	rows := displayRows{}
	rows.RightAlign(false, true, true)
	for _, stats := range dx.sorted {
		rows.addRow(dx.formatExt(stats), dx.formatCount(stats), dx.formatSize(stats))
	}
	return rows
}

func (dx *DirX) formatCount(stats *Stats) string {
	if dx.NoCommas {
		return fmt.Sprintf("%d", stats.count)
	}
	return printer.Sprint(stats.count)
}

func (dx *DirX) formatSmallest(stats *Stats) string {
	if dx.NoCommas {
		return fmt.Sprintf("%d", stats.smallest)
	}
	return printer.Sprint(stats.smallest)
}

func (dx *DirX) formatLargest(stats *Stats) string {
	if dx.NoCommas {
		return fmt.Sprintf("%d", stats.largest)
	}
	return printer.Sprint(stats.largest)
}

func (dx *DirX) formatSize(stats *Stats) string {
	if dx.NoCommas {
		return fmt.Sprintf("%d", stats.bytes)
	}
	return printer.Sprint(stats.bytes)
}

func (dx *DirX) formatExt(stats *Stats) string {
	if dx.ShowSingleName && stats.count == 1 {
		return stats.firstFile
	}
	return "." + stats.ext
}

func getColWidth() (int, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	cols := strings.Split(strings.TrimSpace(string(out)), " ")
	return strconv.Atoi(cols[1])
}
