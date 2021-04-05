package dirx

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type displayRows struct {
	rows      [][]string
	colWidths []int
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
		cols[i] = fmt.Sprintf("%-*s", width, r.rows[index][i])
	}
	return strings.Join(cols, " ")
}

// Len is the standard length interface
func (r *displayRows) Len() int {
	return len(r.rows)
}

// Print prints out the values
func (dx *DirX) Print() {
	width, _ := getColWidth()
	_ = width
	rows := displayRows{}
	for _, stats := range dx.sorted {
		rows.addRow(dx.formatCount(stats), dx.formatExt(stats), dx.formatSize(stats))
	}
	for i := 0; i < rows.Len(); i++ {
		fmt.Println(rows.getRow(i))
	}
}

func (dx *DirX) formatCount(stats Stats) string {
	return strconv.Itoa(stats.count)
}

func (dx *DirX) formatSmallest(stats Stats) string {
	return strconv.FormatInt(stats.smallest, 10)
}

func (dx *DirX) formatLargest(stats Stats) string {
	return strconv.FormatInt(stats.largest, 10)
}

func (dx *DirX) formatSize(stats Stats) string {
	return strconv.FormatInt(stats.bytes, 10)
}

func (dx *DirX) formatExt(stats Stats) string {
	if dx.ShowSingleName && stats.count == 1 {
		return stats.firstFile
	}
	return "*." + stats.ext
}

func getColWidth() (int, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	cols := strings.Split(strings.TrimSpace(string(out)), " ")
	val, err := strconv.Atoi(cols[1])
	return val, nil
}
