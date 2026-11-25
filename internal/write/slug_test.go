package write

import "testing"

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Éléments de l'API — v2": "elements-de-l-api-v2",
		"a.b_c":                  "a-b-c",
		"—":                      "",
		"  Hello   World  ":      "hello-world",
		"日本語":                    "",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocale(t *testing.T) {
	en, fr := Locale("en"), Locale("fr")
	if en.EmptySection != "Not documented." || en.EmptyGroup != "No page." || en.MoveTitle != "Page %s renamed %s" || en.MoveNote != "renamed from %s" {
		t.Fatalf("en = %+v", en)
	}
	if fr.EmptySection != "Non renseigné." || fr.EmptyGroup != "Aucune page." || fr.MoveTitle != "Page %s renommée %s" || fr.MoveNote != "renommé depuis %s" {
		t.Fatalf("fr = %+v", fr)
	}
}
