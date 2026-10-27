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

func TestStripANSI_OSCAndLong(t *testing.T) {
	// OSC title sequence BEL-terminated
	in := "\x1b]0;title\x07hello"
	out := stripANSI(in)
	if out != "hello" {
		t.Fatalf("got %q", out)
	}
	// CSI colors mixed
	in2 := "\x1b[1;32mOK\x1b[0m done"
	out2 := stripANSI(in2)
	if out2 != "OK done" {
		t.Fatalf("got %q", out2)
	}
}

func TestScanLinesCRPartialAtEOF(t *testing.T) {
	sc := bufio.NewScanner(strings.NewReader("no-newline-at-end"))
	sc.Split(scanLinesCR)
	if !sc.Scan() {
		t.Fatal("expected token")
	}
	if sc.Text() != "no-newline-at-end" {
		t.Fatalf("got %q", sc.Text())
	}
	if sc.Scan() {
		t.Fatal("extra token")
	}
}
