package dirx

import (
	"testing"
)

func TestFormat(t *testing.T) {
	sortedList := []*Stats{
		{
			ext:       "bat",
			firstFile: "readme.bat",
			count:     1234,
			bytes:     5678,
			smallest:  0,
			largest:   2345,
		},
	}
	tests := []struct {
		d    *DirX
		want string
	}{
		{
			d: &DirX{
				NoCommas: true,
				sorted:   sortedList,
			},
			want: ".bat 1234 5678",
		}, {
			d: &DirX{
				sorted: sortedList,
			},
			want: ".bat 1,234 5,678",
		},
	}
	for _, test := range tests {
		rows := test.d.formatRows()
		got := rows.getRow(0)
		if got != test.want {
			t.Errorf("geRow(0) = %q, want %q", got, test.want)
		}
	}
}
