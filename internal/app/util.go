package app

import "strings"

// urlencode is PHP urlencode (%XX with + for spaces).
func urlencode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch == '.':
			b.WriteByte(ch)
		case ch == ' ':
			b.WriteByte('+')
		default:
			const hex = "0123456789ABCDEF"
			b.WriteByte('%')
			b.WriteByte(hex[ch>>4])
			b.WriteByte(hex[ch&0xF])
		}
	}
	return b.String()
}
