package app

// Port of Sources/Errors.php. PHP's fatal_error()/redirectexit() never
// return — they render and exit. In Go that is modeled with panic(smfExit{})
// unwinding to the dispatcher, which has already gotten the page into c.Out.

import (
	"regexp"
	"time"
)

// smfExit is panicked to end the request like PHP's exit/die. The page (or
// redirect headers) must already be prepared.
type smfExit struct{}

// exit ends the request now; the dispatcher flushes c.Out.
func (c *Ctx) exit() {
	panic(smfExit{})
}

var sescRe = regexp.MustCompile(`;sesc=[^&;]+`)

// logError is log_error() from Errors.php.
func (c *Ctx) logError(message string) string {
	a := c.App
	if a.SettingEmpty("enableErrorLogging") {
		return message
	}

	// Basically, htmlspecialchars it minus &. (for entities!)
	r := newReplacerChain(message,
		"<", "&lt;", ">", "&gt;", `"`, "&quot;")
	r = newReplacerChain(r,
		"&lt;br /&gt;", "<br />", "&lt;b&gt;", "<b>", "&lt;/b&gt;", "</b>", "\n", "<br />")

	// Don't log the session hash in the url twice, it's a waste.
	queryString := ""
	if c.R != nil && c.R.URL.RawQuery != "" {
		queryString = Htmlspecialchars("?" + sescRe.ReplaceAllString(c.R.URL.RawQuery, ";sesc"))
	}
	if c.POST != nil && c.POST.Has("board") && !c.GET.Has("board") {
		if queryString == "" {
			queryString = "board=" + c.POST.Str("board")
		} else {
			queryString += ";board=" + c.POST.Str("board")
		}
	}

	userID, ip, sc := 0, "", ""
	if c.User != nil {
		userID = c.User.ID
		ip = c.User.IP
	}
	sc = c.Sc

	a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}log_errors
			(ID_MEMBER, logTime, ip, url, message, session)
		VALUES (?, ?, SUBSTR(?, 1, 16), SUBSTR(?, 1, 65534), SUBSTR(?, 1, 65534), ?)`),
		userID, time.Now().Unix(), ip, queryString, r, sc)

	return r
}

func newReplacerChain(s string, pairs ...string) string {
	for i := 0; i+1 < len(pairs); i += 2 {
		s = replaceAll(s, pairs[i], pairs[i+1])
	}
	return s
}

func replaceAll(s, old, new string) string {
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ErrorCtx is the page context for the fatal_error sub template.
type ErrorCtx struct {
	Title   string // $context['error_title']
	Message string // $context['error_message']
}

// fatalError is fatal_error() from Errors.php: render the error page and
// stop.
func (c *Ctx) fatalError(message string, log bool) {
	a := c.App

	errCtx, ok := c.Page.(*ErrorCtx)
	if !ok || errCtx.Title == "" {
		title := c.Txt("106")
		msg := message
		if log || a.Setting("enableErrorLogging") == "2" {
			msg = c.logError(message)
		}
		errCtx = &ErrorCtx{Title: title, Message: msg}
		c.Page = errCtx
	}

	if c.PageTitle == "" {
		c.PageTitle = errCtx.Title
	}

	c.SubTemplate = templateFatalError

	// We want whatever for the header, and a footer. (footer includes sub
	// template!)
	c.obExit(nil, boolPtr(true), false)
}

// fatalLangError is fatal_lang_error(): message comes from the Errors
// language file, with optional sprintf arguments.
func (c *Ctx) fatalLangError(key string, log bool, sprintf ...any) {
	c.loadLanguage("Errors")
	msg := c.Txt(key)
	if len(sprintf) > 0 {
		msg = phpSprintf(msg, sprintf...)
	}
	c.fatalError(msg, log)
}

// fatalDBError is the db_error() path: log + show the database error page.
// The MySQL auto-repair machinery does not apply to SQLite.
func (c *Ctx) fatalDBError(err error) {
	c.loadLanguage("Errors")
	c.logError(c.Txt("1001") + ": " + err.Error())
	title := c.Txt("1001")
	msg := c.Txt("1002")
	if c.allowedTo("admin_forum") {
		msg = Htmlspecialchars(err.Error())
	}
	c.Page = &ErrorCtx{Title: title, Message: msg}
	c.fatalError(msg, false)
}

func boolPtr(b bool) *bool { return &b }

// phpSprintf handles the %s/%d/%1$s subset SMF's language strings use.
func phpSprintf(format string, args ...any) string {
	out := make([]byte, 0, len(format)+32)
	argIdx := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			out = append(out, format[i])
			continue
		}
		j := i + 1
		// Positional: %1$s
		pos := -1
		k := j
		for k < len(format) && format[k] >= '0' && format[k] <= '9' {
			k++
		}
		if k > j && k < len(format) && format[k] == '$' {
			pos = atoi(format[j:k]) - 1
			j = k + 1
		}
		if j >= len(format) {
			out = append(out, format[i])
			continue
		}
		var val any
		switch format[j] {
		case 's', 'd':
			if pos >= 0 && pos < len(args) {
				val = args[pos]
			} else if pos < 0 && argIdx < len(args) {
				val = args[argIdx]
				argIdx++
			}
			switch v := val.(type) {
			case string:
				if format[j] == 'd' {
					out = append(out, itoa(atoi(v))...)
				} else {
					out = append(out, v...)
				}
			case int:
				out = append(out, itoa(v)...)
			case nil:
				// missing arg: PHP warns, emits nothing
			}
			i = j
		case '%':
			out = append(out, '%')
			i = j
		default:
			out = append(out, format[i])
		}
	}
	return string(out)
}
