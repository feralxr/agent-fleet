package ui

import "testing"

func TestExpandTemplate(t *testing.T) {
	body, id, ok := expandTemplate("/review")
	if !ok || id != "review" || body == "" {
		t.Fatalf("review: ok=%v id=%s body=%q", ok, id, body)
	}
	body, id, ok = expandTemplate("/tests cover auth")
	if !ok || id != "tests" {
		t.Fatalf("tests+ctx: ok=%v id=%s", ok, id)
	}
	if body == "cover auth" || body == "" {
		t.Fatalf("expected expanded body with context, got %q", body)
	}
	_, _, ok = expandTemplate("plain prompt")
	if ok {
		t.Fatal("plain should not expand")
	}
	_, _, ok = expandTemplate("/unknown")
	if ok {
		t.Fatal("unknown should not expand")
	}
}

func TestTemplateByID(t *testing.T) {
	body, label, ok := templateByID("commit")
	if !ok || body == "" || label == "" {
		t.Fatalf("commit pin lookup failed")
	}
	_, _, ok = templateByID("nope")
	if ok {
		t.Fatal("expected miss")
	}
}

func TestMergePinSlots(t *testing.T) {
	// Ensure all default templates have unique IDs suitable for pins
	seen := map[string]bool{}
	for _, tmpl := range promptTemplates {
		if seen[tmpl.ID] {
			t.Fatalf("duplicate template id %s", tmpl.ID)
		}
		seen[tmpl.ID] = true
		if tmpl.Body == "" {
			t.Fatalf("empty body for %s", tmpl.ID)
		}
	}
}
