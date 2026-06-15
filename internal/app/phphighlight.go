package app

// Port of highlight_php_code() (Subs.php 2486) and the [code]/[php] validate
// callbacks, including an imitation of PHP 5.6's highlight_string().
//
// highlight_string colors (php.ini defaults):
//   highlight.string  #DD0000   highlight.comment #FF8000
//   highlight.keyword #007700   highlight.default #0000BB
//   highlight.html    #000000
// Keywords/operators/punctuation get keyword color; variables, numbers and
// identifiers get default color. Whitespace stays in the current span.
// Output: <code><span style="color: #000000">\n...</span>\n</code> with
// entities escaped, spaces as &nbsp; and newlines as <br />.
//
// NOTE: exact parity with PHP's tokenizer for exotic code needs the golden
// fixture corpus (PORTING.md); the common cases are handled here.

import (
	"regexp"
	"strings"
)

// highlightPHPSections is the [code] validate callback: only the
// &lt;?php ... ?&gt; sections of the data get highlighted.
func (c *Ctx) highlightPHPSections(data string) string {
	parts := splitKeepDelims(data, []string{"&lt;?php", "?&gt;"})
	for i := 0; i < len(parts); i++ {
		// Do PHP code coloring?
		if parts[i] != "&lt;?php" {
			continue
		}
		phpString := ""
		for i+1 < len(parts) && parts[i] != "?&gt;" {
			phpString += parts[i]
			parts[i] = ""
			i++
		}
		parts[i] = c.highlightPHPCode(phpString + parts[i])
	}

	// Fix the PHP code stuff...
	out := strings.ReplaceAll(strings.Join(parts, ""), "<pre style=\"display: inline;\">\t</pre>", "\t")

	// Older browsers are annoying, aren't they?
	if c.Browser.IsIE4 || c.Browser.IsIE5 || c.Browser.IsIE55 {
		out = strings.ReplaceAll(out, "\t", "<pre style=\"display: inline;\">\t</pre>")
	} else if !c.Browser.IsGecko {
		out = strings.ReplaceAll(out, "\t", "<span style=\"white-space: pre;\">\t</span>")
	}
	return out
}

// highlightPHPBlock is the [php] validate callback: the whole content is
// PHP, wrapped in tags if needed and unwrapped after.
func (c *Ctx) highlightPHPBlock(data string) string {
	addBegin := !strings.HasPrefix(strings.TrimSpace(data), "&lt;?")
	if addBegin {
		data = c.highlightPHPCode("&lt;?php " + data + "?&gt;")
		// Strip the artificial open/close tags back off.
		data = phpOpenStripRe.ReplaceAllString(data, "$1")
		data = phpCloseStripRe.ReplaceAllString(data, "$1")
	} else {
		data = c.highlightPHPCode(data)
	}
	return data
}

var phpOpenStripRe = regexp.MustCompile(`^(.+?)&lt;\?.{0,40}?php(&nbsp;|\s)`)
var phpCloseStripRe = regexp.MustCompile(`\?&gt;((?:</(?:font|span)>)*)$`)

// splitKeepDelims is preg_split with PREG_SPLIT_DELIM_CAPTURE on literal
// delimiters.
func splitKeepDelims(s string, delims []string) []string {
	var parts []string
	for {
		best, bestDelim := -1, ""
		for _, d := range delims {
			if i := strings.Index(s, d); i >= 0 && (best < 0 || i < best) {
				best, bestDelim = i, d
			}
		}
		if best < 0 {
			parts = append(parts, s)
			return parts
		}
		if best > 0 {
			parts = append(parts, s[:best])
		}
		parts = append(parts, bestDelim)
		s = s[best+len(bestDelim):]
	}
}

// highlightPHPCode is highlight_php_code($code).
func (c *Ctx) highlightPHPCode(code string) string {
	// Remove special characters.
	code = unHtmlspecialchars(strings.NewReplacer(
		"<br />", "\n", "\t", "SMF_TAB();", "&#91;", "[").Replace(code))

	buffer := strings.NewReplacer("\n", "", "\r", "").Replace(highlightString(code))

	// Yes, I know this is kludging it, but this is the best way to preserve
	// tabs from PHP :P.
	buffer = smfTabRe.ReplaceAllString(buffer, "<pre style=\"display: inline;\">\t</pre>")

	return strings.NewReplacer("'", "&#039;", "<code>", "", "</code>", "").Replace(buffer)
}

var smfTabRe = regexp.MustCompile(`SMF_TAB(</(?:font|span)><(?:font color|span style)="[^"]*?">)?\(\);`)

// ---- highlight_string imitation ----

const (
	colHTML    = "#000000"
	colDefault = "#0000BB"
	colKeyword = "#007700"
	colString  = "#DD0000"
	colComment = "#FF8000"
)

var phpKeywords = map[string]bool{}

func init() {
	for _, k := range []string{
		"abstract", "and", "array", "as", "break", "case", "catch", "class", "clone",
		"const", "continue", "declare", "default", "do", "echo", "else", "elseif",
		"empty", "enddeclare", "endfor", "endforeach", "endif", "endswitch", "endwhile",
		"eval", "exit", "die", "extends", "final", "for", "foreach", "function",
		"global", "goto", "if", "implements", "include", "include_once", "instanceof",
		"interface", "isset", "list", "namespace", "new", "or", "print", "private",
		"protected", "public", "require", "require_once", "return", "static", "switch",
		"throw", "try", "unset", "use", "var", "while", "xor", "true", "false", "null",
	} {
		phpKeywords[k] = true
	}
}

type hlWriter struct {
	b          strings.Builder
	tokenColor string // color of the open inner token span ("" = none)
}

// emit writes text in the given color. PHP's highlight_string keeps the outer
// #000000 (colHTML) span open as a permanent wrapper and NESTS coloured token
// spans inside it (close-then-open among non-default colours); default-coloured
// text is written raw inside the wrapper.
func (w *hlWriter) emit(color, text string) {
	if text == "" {
		return
	}
	if color == colHTML {
		if w.tokenColor != "" {
			w.b.WriteString("</span>")
			w.tokenColor = ""
		}
		w.b.WriteString(hlEscape(text))
		return
	}
	if w.tokenColor != color {
		if w.tokenColor != "" {
			w.b.WriteString("</span>")
		}
		w.b.WriteString(`<span style="color: ` + color + `">`)
		w.tokenColor = color
	}
	w.b.WriteString(hlEscape(text))
}

// emitRaw writes text in the current token span without switching (whitespace).
func (w *hlWriter) emitRaw(text string) {
	w.b.WriteString(hlEscape(text))
}

// close closes the open token span (if any) and the #000000 wrapper.
func (w *hlWriter) close() {
	if w.tokenColor != "" {
		w.b.WriteString("</span>")
		w.tokenColor = ""
	}
	w.b.WriteString("</span>")
}

// hlEscape mirrors PHP's output escaping in highlight_string: entities, then
// spaces to &nbsp; and newlines to <br />.
func hlEscape(s string) string {
	s = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
	s = strings.ReplaceAll(s, " ", "&nbsp;")
	s = strings.ReplaceAll(s, "\n", "<br />")
	return s
}

// highlightString imitates PHP highlight_string($code, true).
func highlightString(code string) string {
	w := &hlWriter{}
	w.b.WriteString(`<code><span style="color: ` + colHTML + `">` + "\n")

	i := 0
	n := len(code)
	for i < n {
		// Outside PHP: html until an open tag.
		open := strings.Index(code[i:], "<?")
		if open < 0 {
			w.emit(colHTML, code[i:])
			break
		}
		if open > 0 {
			w.emit(colHTML, code[i:i+open])
			i += open
		}

		// Consume the open tag (with trailing whitespace, like the
		// tokenizer's T_OPEN_TAG).
		tagEnd := i + 2
		if strings.HasPrefix(code[tagEnd:], "php") {
			tagEnd += 3
		} else if strings.HasPrefix(code[tagEnd:], "=") {
			tagEnd++
		}
		for tagEnd < n && (code[tagEnd] == ' ' || code[tagEnd] == '\t' || code[tagEnd] == '\n' || code[tagEnd] == '\r') {
			tagEnd++
			break // T_OPEN_TAG eats exactly one whitespace char
		}
		w.emit(colDefault, code[i:tagEnd])
		i = tagEnd

		// Inside PHP until ?>.
		for i < n {
			ch := code[i]
			switch {
			case ch == '?' && i+1 < n && code[i+1] == '>':
				w.emit(colDefault, "?>")
				i += 2
				goto outsidePHP

			case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
				j := i
				for j < n && (code[j] == ' ' || code[j] == '\t' || code[j] == '\n' || code[j] == '\r') {
					j++
				}
				w.emitRaw(code[i:j])
				i = j

			case ch == '/' && i+1 < n && code[i+1] == '/', ch == '#':
				j := i
				for j < n && code[j] != '\n' && !(code[j] == '?' && j+1 < n && code[j+1] == '>') {
					j++
				}
				w.emit(colComment, code[i:j])
				i = j

			case ch == '/' && i+1 < n && code[i+1] == '*':
				j := strings.Index(code[i+2:], "*/")
				if j < 0 {
					j = n
				} else {
					j = i + 2 + j + 2
				}
				w.emit(colComment, code[i:j])
				i = j

			case ch == '\'' || ch == '"':
				quote := ch
				j := i + 1
				for j < n {
					if code[j] == '\\' && j+1 < n {
						j += 2
						continue
					}
					if code[j] == quote {
						j++
						break
					}
					j++
				}
				w.emit(colString, code[i:j])
				i = j

			case ch == '$':
				j := i + 1
				for j < n && (isWordByte(code[j])) {
					j++
				}
				w.emit(colDefault, code[i:j])
				i = j

			case ch >= '0' && ch <= '9':
				j := i
				for j < n && (code[j] >= '0' && code[j] <= '9' || code[j] == '.' || code[j] == 'x' ||
					code[j] >= 'a' && code[j] <= 'f' || code[j] >= 'A' && code[j] <= 'F') {
					j++
				}
				w.emit(colDefault, code[i:j])
				i = j

			case isWordStartByte(ch):
				j := i
				for j < n && isWordByte(code[j]) {
					j++
				}
				word := code[i:j]
				if phpKeywords[strings.ToLower(word)] {
					w.emit(colKeyword, word)
				} else {
					w.emit(colDefault, word)
				}
				i = j

			default:
				// Operators and punctuation: keyword color.
				w.emit(colKeyword, string(ch))
				i++
			}
		}
	outsidePHP:
	}

	w.close()
	return w.b.String() + "\n</code>"
}

func isWordStartByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '_' || b >= 0x80
}

func isWordByte(b byte) bool {
	return isWordStartByte(b) || b >= '0' && b <= '9'
}
