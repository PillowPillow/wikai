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
		{"W| w|", friday, "W| w|"},
		{"[2 HH", friday, "[2 14"},
	} {
		got, err := Format(c.format, c.at, "en")
		if err != nil || got != c.want {
			t.Errorf("Format(%q) = %q, %v; want %q", c.format, got, err, c.want)
		}
	}
}

func day(month time.Month, d int) time.Time {
	return time.Date(2026, month, d, 9, 0, 0, 0, time.UTC)
}

func TestFormatNamesAndOrdinals(t *testing.T) {
	for _, c := range []struct {
		locale, format string
		at             time.Time
		want           string
	}{
		{"en", "dddd ddd dd, MMMM MMM Do", friday, "Friday Fri Fr, January Jan 2nd"},
		{"en", "Do Do Do Do Do Do Do Do", day(1, 1), "1st 1st 1st 1st 1st 1st 1st 1st"},
		{"en", "Do", day(1, 3), "3rd"},
		{"en", "Do", day(1, 4), "4th"},
		{"en", "Do", day(1, 11), "11th"},
		{"en", "Do", day(1, 12), "12th"},
		{"en", "Do", day(1, 13), "13th"},
		{"en", "Do", day(1, 21), "21st"},
		{"en", "Do", day(1, 22), "22nd"},
		{"en", "Do", day(1, 23), "23rd"},
		{"en", "dd", day(1, 4), "Su"},
		{"fr", "dddd ddd dd D MMMM MMM", friday, "vendredi ven. ve 2 janvier janv."},
		{"fr", "Do Do", day(1, 1), "1er 1er"},
		{"fr", "Do", day(1, 2), "2"},
		{"fr", "dddd ddd dd", day(1, 4), "dimanche dim. di"},
		{"fr", "MMMM MMM", day(2, 1), "février févr."},
		{"fr", "MMMM MMM", day(6, 1), "juin juin"},
		{"fr", "MMMM MMM", day(7, 1), "juillet juil."},
		{"fr", "MMMM MMM", day(8, 1), "août août"},
		{"fr", "MMMM MMM", day(9, 1), "septembre sept."},
		{"fr", "MMMM MMM", day(12, 1), "décembre déc."},
		{"fr", "dddd D MMMM YYYY [à] HH:mm A", friday, "vendredi 2 janvier 2026 à 14:05 PM"},
	} {
		got, err := Format(c.format, c.at, c.locale)
		if err != nil || got != c.want {
			t.Errorf("Format(%s, %q) = %q, %v; want %q", c.locale, c.format, got, err, c.want)
		}
	}
}

func TestFormatRefusesMomentTokensOutsideTheSupportedList(t *testing.T) {
	for format, tok := range map[string]string{
		"YYYY-Q": "Q", "DDD": "DDD", "w": "w", "X": "X", "LL": "LL", "l": "l", "Z": "Z", "Mo": "Mo", "e": "e", "YYYYY": "YYYYY", "SSS": "SSS",
	} {
		_, err := Format(format, friday, "en")
		u, ok := err.(*UnsupportedError)
		if !ok || u.Token != tok {
			t.Errorf("Format(%q) error = %v; want unsupported token %q", format, err, tok)
		}
	}
}

func TestFormatKeepsEscapedTokens(t *testing.T) {
	got, err := Format(`[Q] \Q \L`, friday, "en")
	if err != nil || got != "Q Q L" {
		t.Fatalf("got %q, %v", got, err)
	}
}
