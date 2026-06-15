// Package phpser implements the subset of PHP's serialize()/unserialize()
// format that SMF 1.1 uses for its login cookie: a flat array of ints and
// strings with integer keys, e.g.
//
//	a:4:{i:0;i:1234;i:1;s:40:"0123...";i:2;i:1735689600;i:3;i:2;}
package phpser

import (
	"fmt"
	"strconv"
	"strings"
)

// Serialize encodes a flat slice as a PHP array with keys 0..n-1.
// Supported element types: int, int64, string.
func Serialize(values []any) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "a:%d:{", len(values))
	for i, v := range values {
		fmt.Fprintf(&b, "i:%d;", i)
		switch x := v.(type) {
		case int:
			fmt.Fprintf(&b, "i:%d;", x)
		case int64:
			fmt.Fprintf(&b, "i:%d;", x)
		case string:
			fmt.Fprintf(&b, "s:%d:\"%s\";", len(x), x)
		default:
			return "", fmt.Errorf("phpser: unsupported type %T", v)
		}
	}
	b.WriteString("}")
	return b.String(), nil
}

// Unserialize decodes a PHP array of ints and strings with sequential
// integer keys 0..n-1 into a slice. Elements are int64 or string.
func Unserialize(s string) ([]any, error) {
	p := &parser{s: s}
	if !p.literal("a:") {
		return nil, p.err("expected array")
	}
	n, ok := p.intUntil(':')
	if !ok || n < 0 {
		return nil, p.err("bad array length")
	}
	if !p.literal("{") {
		return nil, p.err("expected {")
	}
	out := make([]any, 0, n)
	for i := int64(0); i < n; i++ {
		k, err := p.value()
		if err != nil {
			return nil, err
		}
		ki, isInt := k.(int64)
		if !isInt || ki != i {
			return nil, p.err("non-sequential key")
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if !p.literal("}") {
		return nil, p.err("expected }")
	}
	if p.pos != len(p.s) {
		return nil, p.err("trailing data")
	}
	return out, nil
}

type parser struct {
	s   string
	pos int
}

func (p *parser) err(msg string) error {
	return fmt.Errorf("phpser: %s at offset %d", msg, p.pos)
}

func (p *parser) literal(lit string) bool {
	if strings.HasPrefix(p.s[p.pos:], lit) {
		p.pos += len(lit)
		return true
	}
	return false
}

// intUntil reads a possibly-signed integer terminated by sep, consuming sep.
func (p *parser) intUntil(sep byte) (int64, bool) {
	end := strings.IndexByte(p.s[p.pos:], sep)
	if end < 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(p.s[p.pos:p.pos+end], 10, 64)
	if err != nil {
		return 0, false
	}
	p.pos += end + 1
	return n, true
}

func (p *parser) value() (any, error) {
	switch {
	case p.literal("i:"):
		n, ok := p.intUntil(';')
		if !ok {
			return nil, p.err("bad int")
		}
		return n, nil
	case p.literal("s:"):
		n, ok := p.intUntil(':')
		if !ok || n < 0 {
			return nil, p.err("bad string length")
		}
		if !p.literal("\"") {
			return nil, p.err("expected open quote")
		}
		if p.pos+int(n) > len(p.s) {
			return nil, p.err("string overruns input")
		}
		v := p.s[p.pos : p.pos+int(n)]
		p.pos += int(n)
		if !p.literal("\";") {
			return nil, p.err("expected close quote")
		}
		return v, nil
	default:
		return nil, p.err("unsupported value type")
	}
}
