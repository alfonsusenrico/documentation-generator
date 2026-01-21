package util

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode"
)

func RandID(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func IsHexID(id string) bool {
	if len(id) != 16 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= '0' && r <= '9':
			continue
		case r >= 'a' && r <= 'f':
			continue
		default:
			return false
		}
	}
	return true
}

func SanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "TBD"
	}
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_', r == '.':
			return r
		case r == ' ': 
			return '_'
		default:
			return -1
		}
	}, s)
	s = strings.Trim(s, "_-.")
	if s == "" {
		return "TBD"
	}
	rs := []rune(s)
	if len(rs) > 80 {
		s = string(rs[:80])
	}
	return s
}

func Slugify(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	needsDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if needsDash && b.Len() > 0 {
				b.WriteRune('-')
			}
			b.WriteRune(unicode.ToLower(r))
			needsDash = false
			continue
		}
		switch r {
		case ' ', '-', '_':
			needsDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fallback
	}
	return out
}

func DownloadID(name string) string {
	if name == "" {
		return ""
	}
	ext := filepathExt(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}

func filepathExt(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i:]
		}
		if name[i] == '/' || name[i] == '\\' {
			break
		}
	}
	return ""
}
