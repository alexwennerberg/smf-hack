package app

// Port of the Sources/Subs.php core: timeformat, forum_time, comma_format,
// constructPageIndex, shorten_subject, writeLog, redirectexit, obExit,
// setupThemeContext, template_header/footer, theme_copyright,
// determineTopicClass, create_button.

import (
	"math"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"smf/internal/strftime"
)

// layerFuncs maps template layer names ('main_above', 'main_below', ...) to
// their compiled template functions; tpl_*.go files register them.
var layerFuncs = map[string]func(*Ctx){}

func callLayer(c *Ctx, name string) {
	if f, ok := layerFuncs[name]; ok {
		f(c)
	}
}

// forumTime is forum_time(): now (or ts) plus forum/user offsets.
func (c *Ctx) forumTime(useUserOffset bool, ts int64) int64 {
	if ts == 0 {
		ts = time.Now().Unix()
	}
	offset := float64(c.App.SettingInt("time_offset"))
	if useUserOffset && c.User != nil {
		offset += c.User.TimeOffset
	}
	return ts + int64(offset*3600)
}

// strftimeNames builds day/month names from $txt for the DIY localization
// path in timeformat(). For English these match glibc.
func (c *Ctx) strftimeNames() *strftime.Names {
	n := strftime.English
	days := c.TxtList("days")
	for i := 0; i < 7 && i < len(days.Items); i++ {
		n.Days[i] = days.Items[i]
	}
	daysShort := c.TxtList("days_short")
	for i := 0; i < 7 && i < len(daysShort.Items); i++ {
		n.DaysShort[i] = daysShort.Items[i]
	}
	months := c.TxtList("months")
	for i := 0; i < 12 && i < len(months.Items); i++ {
		n.Months[i] = months.Items[i]
	}
	monthsShort := c.TxtList("months_short")
	for i := 0; i < 12 && i < len(monthsShort.Items); i++ {
		n.MonthsShort[i] = monthsShort.Items[i]
	}
	return &n
}

// timeformat is timeformat($logTime, true): full Today/Yesterday handling.
func (c *Ctx) timeformat(logTime int64) string {
	return c.timeformatFmt(logTime, true, "")
}

// timeformatNoToday is timeformat($logTime, false).
func (c *Ctx) timeformatNoToday(logTime int64) string {
	return c.timeformatFmt(logTime, false, "")
}

func (c *Ctx) timeformatFmt(logTime int64, showToday bool, format string) string {
	a := c.App

	userFormat := c.User.TimeFormat
	if userFormat == "" {
		userFormat = a.Setting("time_format")
	}

	// Offset the time.
	t := logTime + int64((c.User.TimeOffset+float64(a.SettingInt("time_offset")))*3600)
	if t < 0 {
		t = 0
	}
	// PHP's strftime formats the offset-shifted timestamp in the server's
	// local zone; do the same so output matches the PHP reference host.
	tm := time.Unix(t, 0)

	if a.SettingInt("todayMod") >= 1 && showToday && format == "" {
		nowTime := c.forumTime(true, 0)
		now := time.Unix(nowTime, 0)

		// Try to make something of a time format string...
		s := ""
		if strings.Contains(userFormat, "%S") {
			s = ":%S"
		}
		todayFmt := "%I:%M" + s + " %p"
		if strings.Contains(userFormat, "%H") || strings.Contains(userFormat, "%T") {
			todayFmt = "%H:%M" + s
		}

		// Same day of the year, same year.... Today!
		if tm.YearDay() == now.YearDay() && tm.Year() == now.Year() {
			return c.Txt("smf10") + c.timeformatFmt(logTime, false, todayFmt)
		}
		// Yesterday...
		if a.Setting("todayMod") == "2" &&
			((tm.YearDay() == now.YearDay()-1 && tm.Year() == now.Year()) ||
				(now.YearDay() == 1 && tm.Year() == now.Year()-1 && tm.Month() == 12 && tm.Day() == 31)) {
			return c.Txt("smf10b") + c.timeformatFmt(logTime, false, todayFmt)
		}
	}

	str := userFormat
	if format != "" {
		str = format
	}
	// SMF's timeformat() replaces %p itself (Subs.php) with lowercase am/pm
	// based on the offset-shifted hour, before handing the string to strftime —
	// so %p never reaches strftime's uppercase AM/PM path.
	if strings.Contains(str, "%p") {
		ap := "am"
		if tm.Hour() >= 12 {
			ap = "pm"
		}
		str = strings.ReplaceAll(str, "%p", ap)
	}
	return strftime.Format(str, tm, c.strftimeNames())
}

// commaFormat is comma_format() driven by the number_format setting.
func (c *Ctx) commaFormat(number int) string {
	format := c.App.Setting("number_format")
	thousands := ""
	if len(format) >= 5 && format[0] == '1' && strings.Contains(format, "234") {
		thousands = format[1:strings.Index(format, "234")]
	} else if format != "1234" && format != "" {
		// Unparsable -> raw, like the PHP fallback.
		return itoa(number)
	}
	s := itoa(number)
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, thousands)
	if neg {
		out = "-" + out
	}
	return out
}

// constructPageIndex is constructPageIndex() — the « 1 2 [3] ... » markup.
// Returns the html and the normalized start value.
func (c *Ctx) constructPageIndex(baseURL string, start, maxValue, numPerPage int, flexibleStart bool) (string, int) {
	a := c.App

	startInvalid := start < 0
	if startInvalid {
		start = 0
	} else if start >= maxValue {
		rem := maxValue % numPerPage
		if rem == 0 {
			rem = numPerPage
		}
		start = maxValue - rem
		if start < 0 {
			start = 0
		}
	} else {
		start = start - start%numPerPage
		if start < 0 {
			start = 0
		}
	}

	// Non-flexible: escape literal % in the base, then append ;start=%d. The
	// flexible form expects the caller to embed a single %d where the start
	// goes (and to have escaped any other literal % itself), matching PHP's
	// sprintf($base_link, $counter, $page) call.
	baseLink := `<a class="navPages" href="` + strings.ReplaceAll(baseURL, "%", "%%") + `;start=%d">%s</a> `
	if flexibleStart {
		baseLink = `<a class="navPages" href="` + baseURL + `">%s</a> `
	}
	link := func(linkStart int, text string) string {
		return fmt.Sprintf(baseLink, linkStart, text)
	}

	var pageindex string
	if a.SettingEmpty("compactTopicPagesEnable") {
		// Show the left arrow.
		if start == 0 {
			pageindex = " "
		} else {
			pageindex = link(start-numPerPage, "&#171;")
		}
		// Show all the pages.
		displayPage := 1
		counter := 0
		for ; counter < maxValue; counter += numPerPage {
			if start == counter && !startInvalid {
				pageindex += "<b>" + itoa(displayPage) + "</b> "
			} else {
				pageindex += link(counter, itoa(displayPage))
			}
			displayPage++
		}
		// Show the right arrow.
		displayStart := start + numPerPage
		if displayStart > maxValue {
			displayStart = maxValue
		}
		if start != counter-maxValue && !startInvalid {
			if displayStart > counter-numPerPage {
				pageindex += " "
			} else {
				pageindex += link(displayStart, "&#187;")
			}
		}
	} else {
		contiguous := a.SettingInt("compactTopicPagesContiguous")
		pageContiguous := (contiguous - contiguous%2) / 2

		// Show the first page. (>1< ... 6 7 [8] 9 10 ... 15)
		if start > numPerPage*pageContiguous {
			pageindex = link(0, "1")
		}
		// Show the ... after the first page.
		if start > numPerPage*(pageContiguous+1) {
			pageindex += "<b> ... </b>"
		}
		// Show the pages before the current one.
		for n := pageContiguous; n >= 1; n-- {
			if start >= numPerPage*n {
				tmpStart := start - numPerPage*n
				pageindex += link(tmpStart, itoa(tmpStart/numPerPage+1))
			}
		}
		// Show the current page.
		if !startInvalid {
			pageindex += "[<b>" + itoa(start/numPerPage+1) + "</b>] "
		} else {
			pageindex += link(start, itoa(start/numPerPage+1))
		}
		// Show the pages after the current one...
		tmpMaxPages := (maxValue - 1) / numPerPage * numPerPage
		for n := 1; n <= pageContiguous; n++ {
			if start+numPerPage*n <= tmpMaxPages {
				tmpStart := start + numPerPage*n
				pageindex += link(tmpStart, itoa(tmpStart/numPerPage+1))
			}
		}
		// Show the '...' part near the end.
		if start+numPerPage*(pageContiguous+1) < tmpMaxPages {
			pageindex += "<b> ... </b>"
		}
		// Show the last number in the list.
		if start+numPerPage*pageContiguous < tmpMaxPages {
			pageindex += link(tmpMaxPages, itoa(tmpMaxPages/numPerPage+1))
		}
	}
	return pageindex, start
}

// unHtmlspecialchars is un_htmlspecialchars().
func unHtmlspecialchars(s string) string {
	return strings.NewReplacer(
		"&quot;", `"`, "&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&#039;", "'", "&nbsp;", " ").Replace(s)
}

// shortenSubject is shorten_subject(): entity-aware length, ASCII fallback.
func shortenSubject(subject string, length int) string {
	runes := entitySplit(subject)
	if len(runes) <= length {
		return subject
	}
	return strings.Join(runes[:length], "") + "..."
}

// entitySplit splits a string into units counting &entity; as one character,
// mirroring $func['substr']/['strlen'] for the ISO-8859-1 case.
func entitySplit(s string) []string {
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			if j := strings.IndexByte(s[i:], ';'); j > 1 && j <= 8 {
				ent := s[i : i+j+1]
				if isKnownEntity(ent) {
					out = append(out, ent)
					i += j
					continue
				}
			}
		}
		out = append(out, s[i:i+1])
	}
	return out
}

func isKnownEntity(ent string) bool {
	switch ent {
	case "&quot;", "&amp;", "&lt;", "&gt;", "&nbsp;":
		return true
	}
	if strings.HasPrefix(ent, "&#") && len(ent) >= 4 {
		for i := 2; i < len(ent)-1; i++ {
			if ent[i] < '0' || ent[i] > '9' {
				return false
			}
		}
		return true
	}
	return false
}

// writeLog is writeLog() from Subs.php: track who's online.
func (c *Ctx) writeLog(force bool) {
	a := c.App
	now := time.Now().Unix()

	if !c.Theme.Empty("display_who_viewing") && (c.Topic != 0 || c.Board != 0) {
		force = true
		if c.Topic != 0 {
			if int(c.Session.GetInt("last_topic_id")) == c.Topic {
				force = false
			}
			c.Session.Set("last_topic_id", c.Topic)
		}
	}

	// Don't mark them as online more than every so often.
	logTime := c.Session.GetInt("log_time")
	if logTime != 0 && logTime >= now-8 && !force {
		return
	}

	serialized := ""
	if !a.SettingEmpty("who_enabled") {
		// PHP serialize()s $_GET + USER_AGENT; JSON serves the same purpose
		// for our own Who.php port.
		m := map[string]string{}
		c.GET.Values(func(k string, v any) {
			if s, ok := v.(string); ok && k != "sesc" {
				m[k] = s
			}
		})
		m["USER_AGENT"] = c.UserAgent
		b, _ := json.Marshal(m)
		serialized = string(b)
	}

	// Guests use their IP, members use their session ID.
	sessionID := c.Session.ID
	if c.User.IsGuest {
		sessionID = "ip" + c.User.IP
	}

	doDelete := true
	if v, ok := a.cache.Get("log_online-update"); ok {
		doDelete = v.(int64) < now-10
	}
	lastActive := int64(a.SettingInt("lastActive"))

	if logTime != 0 && logTime >= now-lastActive*20 {
		if doDelete {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE logTime < ? AND session != ?`),
				now-lastActive*60, sessionID)
			a.cache.Put("log_online-update", now, 10*time.Second)
		}
		res, _ := a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_online SET logTime = ?, ip = ?, url = ? WHERE session = ?`),
			now, inetAton(c.User.IP), serialized, sessionID)
		if res != nil {
			if n, _ := res.RowsAffected(); n == 0 {
				logTime = 0
				c.Session.Set("log_time", 0)
			}
		}
	} else {
		logTime = 0
	}

	if logTime == 0 {
		if doDelete && c.User.ID != 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE logTime < ? OR ID_MEMBER = ?`),
				now-lastActive*60, c.User.ID)
		} else if doDelete {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE logTime < ?`), now-lastActive*60)
		} else if c.User.ID != 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE ID_MEMBER = ?`), c.User.ID)
		}
		a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_online (session, ID_MEMBER, logTime, ip, url) VALUES (?, ?, ?, ?, ?)`),
			sessionID, c.User.ID, now, inetAton(c.User.IP), serialized)
	}

	// Mark your session as being logged.
	c.Session.Set("log_time", now)

	// Well, they are online now.
	if c.Session.GetInt("timeOnlineUpdated") == 0 {
		c.Session.Set("timeOnlineUpdated", now)
	}

	// Set their login time, if not already done within the last minute.
	if c.User.LastLogin != 0 && c.User.LastLogin < now-60 {
		if now-c.Session.GetInt("timeOnlineUpdated") > 60*15 {
			c.Session.Set("timeOnlineUpdated", now)
		}
		c.User.TotalTimeLoggedIn += now - c.Session.GetInt("timeOnlineUpdated")
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET lastLogin = ?, memberIP = ?, memberIP2 = ?, totalTimeLoggedIn = ? WHERE ID_MEMBER = ?`),
			now, c.User.IP, c.BanCheckIP, c.User.TotalTimeLoggedIn, c.User.ID)
		c.Session.Set("timeOnlineUpdated", now)
	}
}

// inetAton converts a dotted IPv4 to its integer form (MySQL INET_ATON).
func inetAton(ip string) int64 {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return 0
	}
	var n int64
	for _, p := range parts {
		v := atoi(p)
		if v < 0 || v > 255 {
			return 0
		}
		n = n<<8 | int64(v)
	}
	return n
}

// inetNtoa is INET_NTOA(): a packed IPv4 integer back to dotted-quad.
func inetNtoa(n int64) string {
	return itoa(int(n>>24&255)) + "." + itoa(int(n>>16&255)) + "." + itoa(int(n>>8&255)) + "." + itoa(int(n&255))
}

// redirectExit is redirectexit(): Location + exit.
func (c *Ctx) redirectExit(setLocation string) {
	add := !strings.HasPrefix(setLocation, "http://") && !strings.HasPrefix(setLocation, "https://") &&
		!strings.HasPrefix(setLocation, "ftp://") && !strings.HasPrefix(setLocation, "ftps://") &&
		!strings.HasPrefix(setLocation, "about:")
	if add {
		if setLocation != "" {
			setLocation = c.App.ScriptURL + "?" + setLocation
		} else {
			setLocation = c.App.ScriptURL
		}
	}

	c.W.Header().Set("Location", strings.ReplaceAll(setLocation, " ", "%20"))
	c.W.WriteHeader(http.StatusFound)
	c.Out.Reset()
	c.obExitTail()
	c.exit()
}

// obExit is obExit(): render header, sub template and footer, then stop.
// nil pointers mean "default" like PHP's null arguments.
func (c *Ctx) obExit(header, doFooter *bool, fromIndex bool) {
	doHeader := !c.headerDone
	if header != nil {
		doHeader = *header
	}
	footer := doHeader
	if doFooter != nil {
		footer = *doFooter
	}

	if doHeader {
		c.templateHeader()
		c.headerDone = true
	}
	if footer {
		// Just show the footer, then. ($context['sub_template'] defaults to
		// 'main', i.e. whatever the controller's page template is.)
		if c.SubTemplate != nil {
			c.SubTemplate(c)
		}
		if !c.footerDone {
			c.footerDone = true
			c.templateFooter()
		}
	}

	c.obExitTail()

	// Don't exit if we're coming from index.php; that will pass through
	// normally.
	if !fromIndex {
		c.exit()
	}
}

// obExitTail is the always-run part of obExit: remember URL + UA in the
// session for the checkSession()/guest-login flows.
func (c *Ctx) obExitTail() {
	if !strings.Contains(c.RequestURL, "action=dlattach") {
		c.Session.Set("old_url", c.RequestURL)
	}
	c.Session.Set("USER_AGENT", c.UserAgent)
}

// templateHeader is template_header(): headers + the *_above layers.
func (c *Ctx) templateHeader() {
	c.setupThemeContext()

	if !c.NoLastModified {
		h := c.W.Header()
		h.Set("Expires", "Mon, 26 Jul 1997 05:00:00 GMT")
		h.Set("Last-Modified", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	}
	charset := c.CharacterSet
	if charset == "" {
		charset = "ISO-8859-1"
	}
	if c.REQUEST.Has("xml") {
		c.W.Header().Set("Content-Type", "text/xml; charset="+charset)
	} else {
		c.W.Header().Set("Content-Type", "text/html; charset="+charset)
	}

	for _, layer := range c.TemplateLayers {
		callLayer(c, layer+"_above")

		// If the user is banned from posting inform them of it.
		if layer == "main" {
			if ban, ok := c.Session.Get("ban").(map[string]any); ok {
				if cannotPost, banned := ban["cannot_post"].(map[string]any); banned {
					c.O(`
				<div class="windowbg" style="margin: 2ex; padding: 2ex; border: 2px dashed red; color: red;">
					`, phpSprintf(c.Txt("you_are_post_banned"), c.bannedName()))
					if reason, _ := cannotPost["reason"].(string); reason != "" {
						c.O(`
					<div style="padding-left: 4ex; padding-top: 1ex;">`, reason, `</div>`)
					}
					c.O(`
				</div>`)
				}
			}
		}
	}
}

// templateFooter is template_footer(): the *_below layers + copyright check.
func (c *Ctx) templateFooter() {
	c.ShowLoadTime = !c.App.SettingEmpty("timeLoadPageEnable")
	c.LoadTime = fmt.Sprintf("%.3g", time.Since(c.timeStart).Seconds())

	for i := len(c.TemplateLayers) - 1; i >= 0; i-- {
		callLayer(c, c.TemplateLayers[i]+"_below")
	}
}

// themeCopyright is theme_copyright(): emits the footer copyright.
func (c *Ctx) themeCopyright() {
	// PHP randomly checks for unclosed comments; do it deterministically.
	buf := c.Out.String()
	stripped := commentRe.ReplaceAllString(buf, "")
	if strings.Contains(stripped, "<!--") {
		c.O("-->")
	}

	c.O(`
		<span class="smalltext" style="display: inline; visibility: visible; font-family: Verdana, Arial, sans-serif;">`)
	c.O(c.forumCopyright())
	c.O(`
		</span>`)
}

var commentRe = regexp.MustCompile(`(?s)<!--.+?-->`)

// determineTopicClass is determineTopicClass() from Subs.php.
func determineTopicClass(isVeryHot, isHot, isPoll, isLocked, isSticky bool) string {
	class := "normal"
	if isVeryHot {
		class = "veryhot"
	} else if isHot {
		class = "hot"
	}
	if isPoll {
		class += "_poll"
	} else {
		class += "_post"
	}
	if isLocked {
		class += "_locked"
	}
	if isSticky {
		class += "_sticky"
	}
	return class
}

// createButton is create_button() from Subs.php.
func (c *Ctx) createButton(name, alt, label, custom string) string {
	if c.Theme.Empty("use_image_buttons") {
		return c.Txt(alt)
	}
	if !c.Theme.Empty("use_buttons") {
		out := `<img src="` + c.Theme.ImagesURL() + `/buttons/` + name + `" alt="` + c.Txt(alt) + `" ` + custom + ` />`
		if label != "" {
			out += "<b>" + c.Txt(label) + "</b>"
		}
		return out
	}
	return `<img src="` + c.Theme.ImagesURL() + `/` + c.User.Language + `/` + name + `" alt="` + c.Txt(alt) + `" ` + custom + ` />`
}

// strftimeLocal formats a (already offset-shifted) timestamp with strftime
// in the server's local zone — plain strftime($fmt, $ts) in PHP.
func (c *Ctx) strftimeLocal(format string, ts int64) string {
	return strftime.Format(format, time.Unix(ts, 0), c.strftimeNames())
}

// commaFormatFloat is comma_format($number, $decimals).
func (c *Ctx) commaFormatFloat(number float64, decimals int) string {
	// PHP's comma_format uses number_format($n, is_float($n) ? override : 0):
	// a division result that's a whole number is an int in PHP, so it prints
	// with no decimals (0/6000 -> "0", not "0.000").
	if number == math.Trunc(number) {
		decimals = 0
	}
	format := c.App.Setting("number_format")
	decimalSep := "."
	if i := strings.Index(format, "234"); i >= 0 && len(format) > i+3 {
		decimalSep = string(format[i+3])
	}
	neg := number < 0
	if neg {
		number = -number
	}
	scaled := number
	for i := 0; i < decimals; i++ {
		scaled *= 10
	}
	rounded := int64(scaled + 0.5)
	intPart := rounded
	frac := int64(0)
	div := int64(1)
	for i := 0; i < decimals; i++ {
		div *= 10
	}
	intPart = rounded / div
	frac = rounded % div
	out := c.commaFormat(int(intPart))
	if neg {
		out = "-" + out
	}
	if decimals > 0 {
		fracStr := i64toa(frac)
		for len(fracStr) < decimals {
			fracStr = "0" + fracStr
		}
		out += decimalSep + fracStr
	}
	return out
}
