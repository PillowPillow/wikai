package marker

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Port of moment's formattingTokens and localFormattingTokens (src/lib/format/format.js), so a format reads as in Obsidian.
var (
	momentTokens = regexp.MustCompile(`(\[[^\[]*\])|(\\e)|(\\)?(eHHmm|[Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|.)`)
	localTokens  = regexp.MustCompile(`(\[[^\[]*\])|(\\)?(LTS|LT|LL?L?L?|l{1,4})`)
	isToken      = regexp.MustCompile(`^(?:eHHmm|[Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?)$`)
)

type UnsupportedError struct {
	Token, Marker string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("unsupported token %q in %s", e.Token, e.Marker)
}

var layouts = map[string]string{
	"YYYY": "2006", "YY": "06", "M": "1", "MM": "01", "D": "2", "DD": "02",
	"HH": "15", "h": "3", "hh": "03", "m": "4", "mm": "04", "s": "5", "ss": "05", "A": "PM", "a": "pm",
}

func token(tok string, at time.Time, locale string) (string, bool) {
	if l, ok := layouts[tok]; ok {
		return at.Format(l), true
	}
	year, week := at.ISOWeek()
	switch tok {
	case "H":
		return strconv.Itoa(at.Hour()), true
	case "W":
		return strconv.Itoa(week), true
	case "WW":
		return fmt.Sprintf("%02d", week), true
	case "GGGG":
		return fmt.Sprintf("%04d", year), true
	case "Hmm", "Hmmss", "hmm", "hmmss":
		out, _ := token(tok[:1], at, locale)
		out += at.Format("04")
		if len(tok) == 5 {
			out += at.Format("05")
		}
		return out, true
	}
	return "", false
}

func Format(format string, at time.Time, locale string) (string, error) {
	marker := "{{date:" + format + "}}"
	for _, m := range localTokens.FindAllStringSubmatch(format, -1) {
		if m[3] != "" && m[2] == "" {
			return "", &UnsupportedError{m[3], marker}
		}
	}
	var b strings.Builder
	for _, m := range momentTokens.FindAllStringSubmatch(format, -1) {
		switch {
		case m[1] != "":
			b.WriteString(m[1][1 : len(m[1])-1])
		case m[2] != "":
			b.WriteString("e")
		case m[3] != "" || !isToken.MatchString(m[4]):
			b.WriteString(m[4])
		default:
			s, ok := token(m[4], at, locale)
			if !ok {
				return "", &UnsupportedError{m[4], marker}
			}
			b.WriteString(s)
		}
	}
	return b.String(), nil
}

func Fill(text string, values map[string]string, at time.Time, locale string) (string, error) {
	var b strings.Builder
	for {
		i := strings.Index(text, "{{")
		if i < 0 {
			b.WriteString(text)
			return b.String(), nil
		}
		b.WriteString(text[:i])
		name, _, ok := strings.Cut(text[i+2:], "}}")
		if !ok || strings.ContainsAny(name, "{}\r\n") {
			b.WriteByte('{')
			text = text[i+1:]
			continue
		}
		text = text[i+len(name)+4:]
		v, known, err := value(name, values, at, locale)
		if err != nil {
			return "", err
		}
		if !known {
			v = "{{" + name + "}}"
		}
		b.WriteString(v)
	}
}

func value(name string, values map[string]string, at time.Time, locale string) (string, bool, error) {
	if v, ok := values[name]; ok {
		return v, true, nil
	}
	switch name {
	case "date":
		name = "date:YYYY-MM-DD"
	case "time":
		name = "time:HH:mm"
	}
	kind, format, _ := strings.Cut(name, ":")
	if kind != "date" && kind != "time" || !strings.Contains(name, ":") {
		return "", false, nil
	}
	s, err := Format(format, at, locale)
	if u, ok := err.(*UnsupportedError); ok {
		u.Marker = "{{" + name + "}}"
	}
	return s, true, err
}
