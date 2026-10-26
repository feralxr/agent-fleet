package process

import (
	"bufio"
	"strings"
	"testing"
)

func TestScanLinesCR(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a\nb\n", []string{"a", "b"}},
		{"a\r\nb\n", []string{"a", "b"}},
		{"progress\rfinal\n", []string{"progress", "final"}},
		{"only", []string{"only"}},
		{"\n\n", []string{"", ""}},
	}
	for _, tc := range cases {
		sc := bufio.NewScanner(strings.NewReader(tc.in))
		sc.Split(scanLinesCR)
		var got []string
		for sc.Scan() {
			got = append(got, sc.Text())
		}
		if err := sc.Err(); err != nil {
			t.Fatal(err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("in %q: got %v want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("in %q [%d]: %q != %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}
