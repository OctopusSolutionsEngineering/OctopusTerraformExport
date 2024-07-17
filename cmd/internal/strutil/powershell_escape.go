package strutil

import (
	"strings"
	"sync"
	"unicode/utf8"
	_ "unsafe"
)

const hex = "0123456789abcdef"

var buf = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

//go:linkname htmlSafeSet encoding/json.safeSet
var htmlSafeSet [utf8.RuneSelf]bool

// JSON will escape the string provided into a JSON-like format, respecting all escaping. This will return
// the string value already in quotes. Acts like (and copied from) 'json.Marshal' on a string value.
func PowershellEscape(s string) string {
	if len(s) == 0 {
		return `""`
	}
	e := buf.Get().(*strings.Builder)
	e.Grow(2 + len(s))
	e.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] {
				i++
				continue
			}
			if start < i {
				e.WriteString(s[start:i])
			}
			e.WriteByte('`')
			switch b {
			case '`', '"':
				e.WriteByte(b)
			case '\n':
				e.WriteByte('n')
			case '\r':
				e.WriteByte('r')
			case '\t':
				e.WriteByte('t')
			default:
				e.WriteString(`u00`)
				e.WriteByte(hex[b>>4])
				e.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				e.WriteString(s[start:i])
			}
			e.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				e.WriteString(s[start:i])
			}
			e.WriteString(`\u202`)
			e.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		e.WriteString(s[start:])
	}
	e.WriteByte('"')
	r := e.String()
	e.Reset()
	buf.Put(e)
	return r
}
