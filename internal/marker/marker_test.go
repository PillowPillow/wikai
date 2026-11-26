package marker

import (
	"testing"
	"time"
)

var friday = time.Date(2026, 1, 2, 14, 5, 9, 0, time.UTC)

func TestFormatNumericTokensAndLiterals(t *testing.T) {
	for _, c := range []struct {
		format string
		at     time.Time
		want   string
	}{
		{"YYYY-MM-DD HH:mm:ss", friday, "2026-01-02 14:05:09"},
		{"YY M D H m s", friday, "26 1 2 14 5 9"},
		{"h hh A a", friday, "2 02 PM pm"},
		{"h A", time.Date(2026, 1, 2, 0, 30, 0, 0, time.UTC), "12 AM"},
		{"h a", time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC), "12 pm"},
		{"W WW GGGG", friday, "1 01 2026"},
		{"GGGG-[W]WW", time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), "2026-W53"},
		{"YYYY-MM-DDTHH:mm", friday, "2026-01-02T14:05"},
		{"[at] HH [h]", friday, "at 14 h"},
		{`\H\h HH`, friday, "Hh 14"},
		{"Y YYY", friday, "Y 26Y"},
		{"Hmm hmmss", friday, "1405 20509"},
		{"[2 HH", friday, "[2 14"},
	} {
		got, err := Format(c.format, c.at, "en")
		if err != nil || got != c.want {
			t.Errorf("Format(%q) = %q, %v; want %q", c.format, got, err, c.want)
		}
	}
}
