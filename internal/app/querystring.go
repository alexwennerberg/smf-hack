package app

// Port of Sources/QueryString.php.
//
// SMF separates URL arguments with ';' (index.php?action=admin;area=...).
// PHP's parse pipeline there is full of load-bearing quirks which are
// replicated exactly:
//
//  1. the whole raw query string is urldecode()d once ('+' becomes space,
//     %XX decoded — yes, this means a value containing %26 splits as '&')
//  2. ';?' and ';' are replaced with '&', literal "%00" and NUL stripped
//  3. '&word&' / trailing '&word' become '&word=' (valueless flags)
//  4. pairs are split and urldecoded AGAIN (parse_str), '.' and ' ' in key
//     names become '_', and key[]/key[sub] build arrays
//  5. every GET value gets htmlspecialchars(ENT_QUOTES)
//
// Never use r.URL.Query(): Go >= 1.17 rejects ';' separators outright.

import (
	"regexp"
	"strconv"
	"strings"
)

// PArray is a PHP-style ordered array: string keys mapping to either string
// or *PArray, preserving insertion order.
type PArray struct {
	keys []string
	m    map[string]any
}

func NewPArray() *PArray {
	return &PArray{m: make(map[string]any)}
}

func (p *PArray) Set(key string, v any) {
	if _, exists := p.m[key]; !exists {
		p.keys = append(p.keys, key)
	}
	p.m[key] = v
}

// Append adds with the next free integer key, like PHP's $a[] = $v.
func (p *PArray) Append(v any) {
	n := 0
	for _, k := range p.keys {
		if i := atoi(k); isAllDigits(k) && i >= n {
			n = i + 1
		}
	}
	p.Set(itoa(n), v)
}

func (p *PArray) Has(key string) bool {
	_, ok := p.m[key]
	return ok
}

func (p *PArray) Get(key string) any { return p.m[key] }

// Str returns the value as a string; arrays and missing keys give "".
// Mirrors the common PHP pattern of (string) casts on request vars.
func (p *PArray) Str(key string) string {
	if s, ok := p.m[key].(string); ok {
		return s
	}
	return ""
}

// Int is (int) $_REQUEST[key].
func (p *PArray) Int(key string) int { return atoi(p.Str(key)) }

// Arr returns the value as a *PArray, nil if scalar or missing.
func (p *PArray) Arr(key string) *PArray {
	if a, ok := p.m[key].(*PArray); ok {
		return a
	}
	return nil
}

func (p *PArray) Keys() []string { return p.keys }
func (p *PArray) Len() int       { return len(p.keys) }

func (p *PArray) Delete(key string) {
	if _, ok := p.m[key]; ok {
		delete(p.m, key)
		for i, k := range p.keys {
			if k == key {
				p.keys = append(p.keys[:i], p.keys[i+1:]...)
				break
			}
		}
	}
}

// Values iterates in insertion order.
func (p *PArray) Values(f func(key string, v any)) {
	for _, k := range p.keys {
		f(k, p.m[k])
	}
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func itoa(n int) string { return strconv.Itoa(n) }

// phpUrldecode is PHP urldecode(): %XX decoded, '+' becomes space, invalid
// escapes pass through unchanged.
func phpUrldecode(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '+':
			b.WriteByte(' ')
		case s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			b.WriteByte(hexVal(s[i+1])<<4 | hexVal(s[i+2]))
			i += 2
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default:
		return c - 'A' + 10
	}
}

// Htmlspecialchars is PHP htmlspecialchars($s, ENT_QUOTES).
func Htmlspecialchars(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#039;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// HtmlspecialcharsRecursive applies Htmlspecialchars to every string value
// (not keys), like htmlspecialchars__recursive.
func HtmlspecialcharsRecursive(p *PArray) {
	for _, k := range p.keys {
		switch v := p.m[k].(type) {
		case string:
			p.m[k] = Htmlspecialchars(v)
		case *PArray:
			HtmlspecialcharsRecursive(v)
		}
	}
}

func Htmltrim(s string) string {
	return strings.Trim(s, " \t\n\r\x0B\x00\xA0")
}

var valuelessFlagRe = regexp.MustCompile(`&(\w+)(?:&|$)`)

// ParseQueryString implements steps 1–4 above, returning the equivalent of
// $_GET (before htmlspecialchars).
func ParseQueryString(raw string) *PArray {
	// Step 1: decode the whole thing once.
	qs := phpUrldecode(raw)
	// Step 2: ';' to '&', strip "%00" text and NUL bytes.
	qs = strings.NewReplacer(";?", "&", ";", "&", "%00", "", "\x00", "").Replace(qs)
	// Step 3: give valueless flags an '='. The PHP regex uses a lookahead so
	// adjacent matches both fix up; emulate by running the replace twice.
	for i := 0; i < 2; i++ {
		qs = valuelessFlagRe.ReplaceAllString(qs, "&$1=&")
	}
	qs = strings.TrimSuffix(qs, "&")
	return parseStr(qs)
}

// parseStr implements PHP parse_str: split on '&', urldecode keys and
// values, mangle '.' and ' ' to '_' in key names, build nested arrays from
// key[...] syntax.
func parseStr(qs string) *PArray {
	out := NewPArray()
	for _, pair := range strings.Split(qs, "&") {
		if pair == "" {
			continue
		}
		rawKey, rawVal, _ := strings.Cut(pair, "=")
		key := phpUrldecode(rawKey)
		val := phpUrldecode(rawVal)
		if key == "" {
			continue
		}

		// Split into base name and bracket path: a[b][] -> "a", ["b", ""].
		base := key
		var path []string
		if br := strings.IndexByte(key, '['); br >= 0 {
			base = key[:br]
			rest := key[br:]
			for len(rest) > 0 && rest[0] == '[' {
				end := strings.IndexByte(rest, ']')
				if end < 0 {
					// Unterminated bracket: PHP treats the rest as one index.
					path = append(path, rest[1:])
					rest = ""
					break
				}
				path = append(path, rest[1:end])
				rest = rest[end+1:]
			}
		}
		// PHP variable-name mangling on the base name only.
		base = strings.NewReplacer(".", "_", " ", "_").Replace(base)

		setNested(out, base, path, val)
	}
	return out
}

// setNested assigns arr[key][path[0]][path[1]]... = val, creating arrays as
// it goes. An empty key means append ($a[] = ...).
func setNested(arr *PArray, key string, path []string, val string) {
	if len(path) == 0 {
		if key == "" {
			arr.Append(val)
		} else {
			arr.Set(key, val)
		}
		return
	}
	var child *PArray
	if key == "" {
		child = NewPArray()
		arr.Append(child)
	} else {
		var ok bool
		child, ok = arr.Get(key).(*PArray)
		if !ok {
			child = NewPArray()
			arr.Set(key, child)
		}
	}
	setNested(child, path[0], path[1:], val)
}
