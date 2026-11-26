package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unicode"
)

type Record struct {
	TS       string   `json:"ts"`
	Raw      string   `json:"raw"`
	SHA256   string   `json:"sha256"`
	Status   string   `json:"status"`
	Pipeline string   `json:"pipeline"`
	Pages    []string `json:"pages"`
	Note     string   `json:"note"`
}

const bom = "\ufeff"

var blockID = regexp.MustCompile(`(?m)[ \t]\^[A-Za-z0-9-]+[ \t]*$`)

func isSpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

func Normalize(text string) string {
	text = strings.TrimLeft(text, bom)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if strings.HasPrefix(text, "---\n") {
		if i := strings.Index(text[3:], "\n---"); i >= 0 {
			text = text[3+i+4:]
		}
	}
	text = blockID.ReplaceAllString(text, "")
	return strings.Join(strings.FieldsFunc(text, isSpace), " ")
}

func Hash(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(Normalize(string(data))))
	return hex.EncodeToString(sum[:]), nil
}

func (r Record) Line() string {
	if r.Pages == nil {
		r.Pages = []string{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func Status(manifestFile, raw, digest string) (string, error) {
	data, err := os.ReadFile(manifestFile)
	if os.IsNotExist(err) {
		return "new", nil
	}
	if err != nil {
		return "", err
	}
	var last *Record
	seenRaw := false
	for i, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return "", fmt.Errorf("%s: line %d: %w", manifestFile, i+1, err)
		}
		if r.SHA256 == digest {
			last = &r
		}
		if r.Raw == raw {
			seenRaw = true
		}
	}
	switch {
	case last != nil:
		return fmt.Sprintf("unchanged %s %s", last.Status, last.Raw), nil
	case seenRaw:
		return "modified", nil
	}
	return "new", nil
}

func Append(file string, r Record) (line string, err error) {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	line = r.Line()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return "", err
	}
	return line, nil
}
