package ui

import "testing"

func TestFilterPalette(t *testing.T) {
	all := filterPalette("")
	if len(all) < 10 {
		t.Fatalf("expected many actions, got %d", len(all))
	}
	got := filterPalette("mod")
	if len(got) == 0 {
		t.Fatal("expected model match")
	}
	found := false
	for _, a := range got {
		if a.ID == "model" {
			found = true
		}
	}
	if !found {
		t.Fatalf("model not in %v", got)
	}
	none := filterPalette("zzzznotfound")
	if len(none) != 0 {
		t.Fatal(none)
	}
}
