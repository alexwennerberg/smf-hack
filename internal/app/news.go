package app

// Port of Sources/News.php: the XML feeds (?action=.xml) in smf / rss / rss2 /
// atom / rdf formats, for the recent/news/members/profile sub-actions, plus
// the recursive dumpTags emitter and cdata_parse. fix_possible_url's
// queryless-URL rewrite is a no-op here (queryless_urls is off by default);
// the guest feed cache is skipped (output identical, just recomputed).

import (
	"strings"
	"time"
)

func init() {
	registerAction(".xml", (*Ctx).ShowXmlFeed)
}

// xmlField is one ordered element in a feed: either a leaf (val) or a nested
// array (sub, arr=true). PHP associative arrays preserve order, so we must too.
type xmlField struct {
	key string
	val string
	sub []xmlField
	arr bool
}

func xleaf(key, val string) xmlField              { return xmlField{key: key, val: val} }
func xnode(sub ...xmlField) xmlField              { return xmlField{sub: sub, arr: true} }
func xnodeK(key string, sub ...xmlField) xmlField { return xmlField{key: key, sub: sub, arr: true} }

// ShowXmlFeed is ShowXmlFeed(): the ?action=.xml dispatcher.
func (c *Ctx) ShowXmlFeed() {
	a := c.App
	scripturl := a.ScriptURL

	if a.SettingEmpty("xmlnews_enable") {
		c.exit()
	}

	c.loadLanguage("Stats")

	// Default to latest 5. No more than 255, please.
	limit := c.GET.Int("limit")
	if !c.GET.Has("limit") || c.GET.Int("limit") < 1 {
		limit = 5
	} else if limit > 255 {
		limit = 255
	}

	maxMsgID := a.SettingInt("maxMsgID")
	totalMessages := a.SettingInt("totalMessages")

	var queryThisBoard, feedTitleSuffix string

	switch {
	case !empty(c.REQUEST.Str("c")) && c.Board == 0:
		var cats []string
		for _, cv := range strings.Split(c.REQUEST.Str("c"), ",") {
			cats = append(cats, itoa(atoi(cv)))
		}
		if len(cats) == 1 {
			var name string
			a.DB.QueryRow(a.Q(`SELECT name FROM {$db_prefix}categories WHERE ID_CAT = ` + cats[0])).Scan(&name)
			feedTitleSuffix = " - " + stripTags(name)
		}
		boards, totalCatPosts := c.recentBoardsByCat(cats)
		if len(boards) != 0 {
			queryThisBoard = "b.ID_BOARD IN (" + joinInts(boards) + ")"
		}
		if totalCatPosts > 100 && totalCatPosts > totalMessages/15 {
			queryThisBoard += "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-400-limit*5))
		}

	case !empty(c.REQUEST.Str("boards")):
		var want []string
		for _, bv := range strings.Split(c.REQUEST.Str("boards"), ",") {
			want = append(want, itoa(atoi(bv)))
		}
		boards, names, totalPosts := c.xmlBoardsByID(want)
		if len(boards) == 0 {
			c.fatalLangError("smf232", true)
		}
		if len(want) == 1 && len(names) == 1 {
			feedTitleSuffix = " - " + stripTags(names[0])
		}
		queryThisBoard = "b.ID_BOARD IN (" + joinInts(boards) + ")"
		if totalPosts > 100 && totalPosts > totalMessages/12 {
			queryThisBoard += "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-500-limit*5))
		}

	case c.Board != 0:
		var totalPosts int
		a.DB.QueryRow(a.Q(`SELECT numPosts FROM {$db_prefix}boards WHERE ID_BOARD = ? LIMIT 1`), c.Board).Scan(&totalPosts)
		if c.BoardInfo != nil {
			feedTitleSuffix = " - " + stripTags(c.BoardInfo.Name)
		}
		queryThisBoard = "b.ID_BOARD = " + itoa(c.Board)
		if totalPosts > 80 && totalPosts > totalMessages/10 {
			queryThisBoard += "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-600-limit*5))
		}

	default:
		recycle := ""
		if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
			recycle = "\n\t\t\tAND b.ID_BOARD != " + itoa(a.SettingInt("recycle_board"))
		}
		queryThisBoard = c.User.QuerySeeBoard + recycle + "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-100-limit*5))
	}

	// Show in rss or proprietary format?
	format := c.GET.Str("type")
	switch format {
	case "smf", "rss", "rss2", "atom", "rdf":
	default:
		format = "smf"
	}

	sa := c.GET.Str("sa")
	subTags := map[string]string{"recent": "recent-post", "news": "article", "members": "member", "profile": ""}
	if _, ok := subTags[sa]; !ok {
		sa = "recent"
	}

	var xml []xmlField
	switch sa {
	case "news":
		xml = c.getXmlNews(queryThisBoard, limit, format)
	case "members":
		xml = c.getXmlMembers(limit, format)
	case "profile":
		xml = c.getXmlProfile(format)
	default:
		xml = c.getXmlRecent(queryThisBoard, limit, format)
	}

	feedTitle := Htmlspecialchars(stripTags(a.Config.MbName)) + feedTitleSuffix
	charset := c.CharacterSet
	if charset == "" {
		charset = "ISO-8859-1"
	}
	langLocale := strings.ReplaceAll(c.Txt("lang_locale"), "_", "-")
	rssDesc := stripTags(c.Txt("xml_rss_desc"))

	// Content-Type header per format.
	switch {
	case format == "smf" || c.REQUEST.Has("debug"):
		c.W.Header().Set("Content-Type", "text/xml; charset="+charset)
	case format == "rss" || format == "rss2":
		c.W.Header().Set("Content-Type", "application/rss+xml; charset="+charset)
	case format == "atom":
		c.W.Header().Set("Content-Type", "application/atom+xml; charset="+charset)
	case format == "rdf":
		c.W.Header().Set("Content-Type", "application/rdf+xml; charset="+charset)
	}

	c.O(`<?xml version="1.0" encoding="`, charset, `"?>`)

	switch {
	case format == "rss" || format == "rss2":
		version := `"0.92"`
		if format == "rss2" {
			version = `"2.0"`
		}
		c.O(`
<rss version=`, version, ` xml:lang="`, langLocale, `">
	<channel>
		<title>`, feedTitle, `</title>
		<link>`, scripturl, `</link>
		<description><![CDATA[`, rssDesc, `]]></description>`)
		c.dumpTags(xml, 2, "item", format)
		c.O(`
	</channel>
</rss>`)

	case format == "atom":
		c.O(`
<feed version="0.3" xmlns="http://purl.org/atom/ns#">
	<title>`, feedTitle, `</title>
	<link rel="alternate" type="text/html" href="`, scripturl, `" />

	<modified>`, time.Now().UTC().Format("2006-01-02T15:04:05Z"), `</modified>
	<tagline><![CDATA[`, rssDesc, `]]></tagline>
	<generator>SMF</generator>
	<author>
		<name>`, stripTags(a.Config.MbName), `</name>
	</author>`)
		c.dumpTags(xml, 2, "entry", format)
		c.O(`
</feed>`)

	case format == "rdf":
		c.O(`
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns="http://purl.org/rss/1.0/">
	<channel rdf:about="`, scripturl, `">
		<title>`, feedTitle, `</title>
		<link>`, scripturl, `</link>
		<description><![CDATA[`, rssDesc, `]]></description>
		<items>
			<rdf:Seq>`)
		for _, item := range xml {
			c.O(`
				<rdf:li rdf:resource="`, fixPossibleURL(c, childVal(item.sub, "link")), `" />`)
		}
		c.O(`
			</rdf:Seq>
		</items>
	</channel>
`)
		c.dumpTags(xml, 1, "item", format)
		c.O(`
</rdf:RDF>`)

	default:
		c.O(`
<smf:xml-feed xmlns:smf="http://www.simplemachines.org/" xmlns="http://www.simplemachines.org/xml/`, sa, `" xml:lang="`, langLocale, `">`)
		c.dumpTags(xml, 1, subTags[sa], format)
		c.O(`
</smf:xml-feed>`)
	}

	c.exit()
}

// dumpTags is dumpTags(): recursively emit the ordered feed data.
func (c *Ctx) dumpTags(fields []xmlField, indent int, tag, format string) {
	for _, f := range fields {
		// Skip it, it's been set to null/empty (PHP's $val == null).
		if f.arr {
			if len(f.sub) == 0 {
				continue
			}
		} else if f.val == "" {
			continue
		}

		key := tag
		if key == "" {
			key = f.key
		}

		c.O("\n", strings.Repeat("\t", indent))

		// atom's link is a self-closing alternate link.
		if format == "atom" && key == "link" {
			c.O(`<link rel="alternate" type="text/html" href="`, fixPossibleURL(c, f.val), `" />`)
			continue
		}

		// Beginning tag.
		if format == "rdf" && key == "item" && f.arr && hasChild(f.sub, "link") {
			c.O(`<`, key, ` rdf:about="`, fixPossibleURL(c, childVal(f.sub, "link")), `">`)
			c.O("\n", strings.Repeat("\t", indent+1), `<dc:format>text/html</dc:format>`)
		} else if format == "atom" && key == "summary" {
			c.O(`<`, key, ` type="html">`)
		} else {
			c.O(`<`, key, `>`)
		}

		if f.arr {
			c.dumpTags(f.sub, indent+1, "", format)
			c.O("\n", strings.Repeat("\t", indent), `</`, key, `>`)
		} else if strings.Contains(f.val, "\n") || strings.Contains(f.val, "<br />") {
			c.O("\n", fixPossibleURL(c, f.val), "\n", strings.Repeat("\t", indent), `</`, key, `>`)
		} else {
			c.O(fixPossibleURL(c, f.val), `</`, key, `>`)
		}
	}
}

func hasChild(sub []xmlField, key string) bool {
	for _, f := range sub {
		if f.key == key {
			return true
		}
	}
	return false
}

func childVal(sub []xmlField, key string) string {
	for _, f := range sub {
		if f.key == key {
			return f.val
		}
	}
	return ""
}

// fixPossibleURL is fix_possible_url(). queryless_urls is off by default, so
// this is the identity in this port (the .html rewrite is not implemented).
func fixPossibleURL(c *Ctx, val string) string {
	return val
}

// cdataParse is cdata_parse($data) with ns=” (the only form News.php uses):
// wrap in CDATA but pull recognized entities (&amp; &lt; &gt; &quot; and
// numeric) back out so they survive as real XML entities.
func cdataParse(data string) string {
	var b strings.Builder
	b.WriteString("<![CDATA[")
	n := len(data)
	pos := 0
	for pos < n {
		amp := strings.IndexByte(data[pos:], '&')
		brk := strings.IndexByte(data[pos:], ']')
		next := n
		if amp >= 0 && pos+amp < next {
			next = pos + amp
		}
		if brk >= 0 && pos+brk < next {
			next = pos + brk
		}

		old := pos
		pos = next
		if pos-old > 0 {
			b.WriteString(data[old:pos])
		}
		if pos >= n {
			break
		}

		switch data[pos] {
		case ']':
			b.WriteString("]]>&#093;<![CDATA[")
			pos++
		case '&':
			rel := strings.IndexByte(data[pos:], ';')
			pos2 := n
			if rel >= 0 {
				pos2 = pos + rel
			}
			ent := substr(data, pos+1, pos2-pos-1)
			if pos+1 < n && data[pos+1] == '#' {
				b.WriteString("]]>" + substr(data, pos, pos2-pos+1) + "<![CDATA[")
			} else if ent == "amp" || ent == "lt" || ent == "gt" || ent == "quot" {
				b.WriteString("]]>" + substr(data, pos, pos2-pos+1) + "<![CDATA[")
			}
			pos = pos2 + 1
		}
	}
	b.WriteString("]]>")
	return strings.ReplaceAll(b.String(), "<![CDATA[]]>", "")
}

// gmdateRSS is gmdate('D, d M Y H:i:s \G\M\T', ts).
func gmdateRSS(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

// gmAtom is gmstrftime('%Y-%m-%dT%H:%M:%SZ', ts).
func gmAtom(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("2006-01-02T15:04:05Z")
}

// xmlBoardsByID is the boards branch's variant that also returns names.
func (c *Ctx) xmlBoardsByID(want []string) ([]int, []string, int) {
	a := c.App
	var boards []int
	var names []string
	total := 0
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.numPosts, b.name
		FROM {$db_prefix}boards AS b
		WHERE b.ID_BOARD IN (` + strings.Join(want, ", ") + `)
			AND ` + c.User.QuerySeeBoard + `
		LIMIT ` + itoa(len(want))))
	if err == nil {
		for rows.Next() {
			var id, numPosts int
			var name string
			rows.Scan(&id, &numPosts, &name)
			boards = append(boards, id)
			names = append(names, name)
			total += numPosts
		}
		rows.Close()
	}
	return boards, names, total
}

// xmlTruncateBody applies the xmlnews_maxlen limit before BBC parsing.
func (c *Ctx) xmlTruncateBody(body string) string {
	maxlen := c.App.SettingInt("xmlnews_maxlen")
	if maxlen == 0 {
		return body
	}
	plain := strings.ReplaceAll(body, "<br />", "\n")
	if len(plain) > maxlen {
		return strings.ReplaceAll(substr(plain, 0, maxlen-3), "\n", "<br />") + "..."
	}
	return body
}

// xmlAuthorEmail decides the rss <author> value (email) or "" to omit it.
func (c *Ctx) xmlAuthorEmail(hideEmail int, email string) string {
	a := c.App
	if (!a.SettingEmpty("guest_hideContacts") && c.User.IsGuest) ||
		(hideEmail != 0 && !a.SettingEmpty("allow_hideEmail") && !c.allowedTo("moderate_forum")) {
		return ""
	}
	return email
}

// getXmlMembers is getXmlMembers(): the most recent members.
func (c *Ctx) getXmlMembers(limit int, format string) []xmlField {
	a := c.App
	scripturl := a.ScriptURL
	if !c.allowedTo("view_mlist") {
		return nil
	}

	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, memberName, realName, dateRegistered, lastLogin
		FROM {$db_prefix}members
		ORDER BY ID_MEMBER DESC
		LIMIT ` + itoa(limit)))
	if err != nil {
		return nil
	}
	var data []xmlField
	for rows.Next() {
		var id int
		var memberName, realName string
		var dateReg, lastLogin int64
		rows.Scan(&id, &memberName, &realName, &dateReg, &lastLogin)
		profileURL := scripturl + "?action=profile;u=" + itoa(id)

		switch {
		case format == "rss" || format == "rss2":
			data = append(data, xnode(
				xleaf("title", cdataParse(realName)),
				xleaf("link", profileURL),
				xleaf("comments", scripturl+"?action=pm;sa=send;u="+itoa(id)),
				xleaf("pubDate", gmdateRSS(dateReg)),
				xleaf("guid", profileURL),
			))
		case format == "rdf":
			data = append(data, xnode(
				xleaf("title", cdataParse(realName)),
				xleaf("link", profileURL),
			))
		case format == "atom":
			data = append(data, xnode(
				xleaf("title", cdataParse(realName)),
				xleaf("link", profileURL),
				xleaf("created", gmAtom(dateReg)),
				xleaf("issued", gmAtom(dateReg)),
				xleaf("modified", gmAtom(lastLogin)),
				xleaf("id", profileURL),
			))
		default:
			data = append(data, xnode(
				xleaf("name", cdataParse(realName)),
				xleaf("time", stripTags(c.timeformat(dateReg))),
				xleaf("id", itoa(id)),
				xleaf("link", profileURL),
			))
		}
	}
	rows.Close()
	return data
}

// getXmlNews is getXmlNews(): the latest first-posts (forum news).
func (c *Ctx) getXmlNews(queryThisBoard string, limit int, format string) []xmlField {
	a := c.App
	scripturl := a.ScriptURL

	boardCond := "t.ID_BOARD"
	if c.Board != 0 {
		boardCond = itoa(c.Board) + "\n\t\t\tAND t.ID_BOARD = " + itoa(c.Board)
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT
			m.smileysEnabled, m.posterTime, m.ID_MSG, m.subject, m.body, t.ID_TOPIC, t.ID_BOARD,
			b.name AS bname, t.numReplies, m.ID_MEMBER, IFNULL(mem.realName, m.posterName) AS posterName,
			mem.hideEmail, IFNULL(mem.emailAddress, m.posterEmail) AS posterEmail, m.modifiedTime
		FROM ({$db_prefix}topics AS t, {$db_prefix}messages AS m, {$db_prefix}boards AS b)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE b.ID_BOARD = ` + boardCond + `
			AND m.ID_MSG = t.ID_FIRST_MSG
			AND ` + queryThisBoard + `
		ORDER BY t.ID_FIRST_MSG DESC
		LIMIT ` + itoa(limit)))
	if err != nil {
		return nil
	}
	var data []xmlField
	for rows.Next() {
		var smileys, idMsg, idTopic, idBoard, numReplies, idMember, hideEmail int
		var posterTime, modifiedTime int64
		var subject, body, bname, posterName, posterEmail string
		rows.Scan(&smileys, &posterTime, &idMsg, &subject, &body, &idTopic, &idBoard,
			&bname, &numReplies, &idMember, &hideEmail, &posterEmail, &modifiedTime)

		body = c.xmlTruncateBody(body)
		body = c.parseBBCCached(body, smileys != 0, itoa(idMsg))
		body = c.censorText(body)
		subject = c.censorText(subject)

		topicURL := scripturl + "?topic=" + itoa(idTopic) + ".0"
		msgGUID := scripturl + "?topic=" + itoa(idTopic) + ".msg" + itoa(idMsg) + "#msg" + itoa(idMsg)

		switch {
		case format == "rss" || format == "rss2":
			item := []xmlField{
				xleaf("title", cdataParse(subject)),
				xleaf("link", topicURL),
				xleaf("description", cdataParse(body)),
			}
			if email := c.xmlAuthorEmail(hideEmail, posterEmail); email != "" {
				item = append(item, xleaf("author", email))
			}
			item = append(item,
				xleaf("comments", scripturl+"?action=post;topic="+itoa(idTopic)+".0"),
				xleaf("category", "<![CDATA["+bname+"]]>"),
				xleaf("pubDate", gmdateRSS(posterTime)),
				xleaf("guid", msgGUID),
			)
			data = append(data, xnode(item...))
		case format == "rdf":
			data = append(data, xnode(
				xleaf("title", cdataParse(subject)),
				xleaf("link", topicURL),
				xleaf("description", cdataParse(body)),
			))
		case format == "atom":
			modTime := modifiedTime
			if modTime == 0 {
				modTime = posterTime
			}
			data = append(data, xnode(
				xleaf("title", cdataParse(subject)),
				xleaf("link", topicURL),
				xleaf("summary", cdataParse(body)),
				xnodeK("author", xleaf("name", posterName)),
				xleaf("created", gmAtom(posterTime)),
				xleaf("issued", gmAtom(posterTime)),
				xleaf("modified", gmAtom(modTime)),
				xleaf("id", msgGUID),
			))
		default:
			posterLink := ""
			if idMember != 0 {
				posterLink = scripturl + "?action=profile;u=" + itoa(idMember)
			}
			data = append(data, xnode(
				xleaf("time", stripTags(c.timeformat(posterTime))),
				xleaf("id", itoa(idMsg)),
				xleaf("subject", cdataParse(subject)),
				xleaf("body", cdataParse(body)),
				xnodeK("poster",
					xleaf("name", cdataParse(posterName)),
					xleaf("id", itoa(idMember)),
					xleaf("link", posterLink)),
				xleaf("topic", itoa(idTopic)),
				xnodeK("board",
					xleaf("name", cdataParse(bname)),
					xleaf("id", itoa(idBoard)),
					xleaf("link", scripturl+"?board="+itoa(idBoard)+".0")),
				xleaf("link", topicURL),
			))
		}
	}
	rows.Close()
	return data
}

// getXmlRecent is getXmlRecent(): the most recent posts.
func (c *Ctx) getXmlRecent(queryThisBoard string, limit int, format string) []xmlField {
	a := c.App
	scripturl := a.ScriptURL

	boardCond := "b.ID_BOARD"
	if c.Board != 0 {
		boardCond = itoa(c.Board) + "\n\t\t\tAND b.ID_BOARD = " + itoa(c.Board)
	}
	var messages []int
	rows, err := a.DB.Query(a.Q(`
		SELECT m.ID_MSG
		FROM ({$db_prefix}messages AS m, {$db_prefix}boards AS b)
		WHERE m.ID_BOARD = ` + boardCond + `
			AND ` + queryThisBoard + `
		ORDER BY m.ID_MSG DESC
		LIMIT ` + itoa(limit)))
	if err == nil {
		for rows.Next() {
			var id int
			rows.Scan(&id)
			messages = append(messages, id)
		}
		rows.Close()
	}
	if len(messages) == 0 {
		return nil
	}

	boardCond2 := "t.ID_BOARD"
	if c.Board != 0 {
		boardCond2 = itoa(c.Board) + "\n\t\t\tAND t.ID_BOARD = " + itoa(c.Board)
	}
	var data []xmlField
	prows, err := a.DB.Query(a.Q(`
		SELECT
			m.smileysEnabled, m.posterTime, m.ID_MSG, m.subject, m.body, m.ID_TOPIC, t.ID_BOARD,
			b.name AS bname, t.numReplies, m.ID_MEMBER, mf.ID_MEMBER AS ID_FIRST_MEMBER,
			IFNULL(mem.realName, m.posterName) AS posterName, mf.subject AS firstSubject,
			IFNULL(memf.realName, mf.posterName) AS firstPosterName, mem.hideEmail,
			IFNULL(mem.emailAddress, m.posterEmail) AS posterEmail, m.modifiedTime
		FROM ({$db_prefix}messages AS m, {$db_prefix}messages AS mf, {$db_prefix}topics AS t, {$db_prefix}boards AS b)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
			LEFT JOIN {$db_prefix}members AS memf ON (memf.ID_MEMBER = mf.ID_MEMBER)
		WHERE t.ID_TOPIC = m.ID_TOPIC
			AND b.ID_BOARD = ` + boardCond2 + `
			AND mf.ID_MSG = t.ID_FIRST_MSG
			AND m.ID_MSG IN (` + joinInts(messages) + `)
		ORDER BY m.ID_MSG DESC
		LIMIT ` + itoa(limit)))
	if err != nil {
		return nil
	}
	for prows.Next() {
		var smileys, idMsg, idTopic, idBoard, numReplies, idMember, idFirstMember, hideEmail int
		var posterTime, modifiedTime int64
		var subject, body, bname, posterName, firstSubject, firstPosterName, posterEmail string
		prows.Scan(&smileys, &posterTime, &idMsg, &subject, &body, &idTopic, &idBoard,
			&bname, &numReplies, &idMember, &idFirstMember, &posterName, &firstSubject,
			&firstPosterName, &hideEmail, &posterEmail, &modifiedTime)

		body = c.xmlTruncateBody(body)
		body = c.parseBBCCached(body, smileys != 0, itoa(idMsg))
		body = c.censorText(body)
		subject = c.censorText(subject)

		msgURL := scripturl + "?topic=" + itoa(idTopic) + ".msg" + itoa(idMsg) + "#msg" + itoa(idMsg)

		switch {
		case format == "rss" || format == "rss2":
			item := []xmlField{
				xleaf("title", cdataParse(subject)),
				xleaf("link", msgURL),
				xleaf("description", cdataParse(body)),
			}
			if email := c.xmlAuthorEmail(hideEmail, posterEmail); email != "" {
				item = append(item, xleaf("author", email))
			}
			item = append(item,
				xleaf("category", cdataParse(bname)),
				xleaf("comments", scripturl+"?action=post;topic="+itoa(idTopic)+".0"),
				xleaf("pubDate", gmdateRSS(posterTime)),
				xleaf("guid", msgURL),
			)
			data = append(data, xnode(item...))
		case format == "rdf":
			data = append(data, xnode(
				xleaf("title", cdataParse(subject)),
				xleaf("link", msgURL),
				xleaf("description", cdataParse(body)),
			))
		case format == "atom":
			modTime := modifiedTime
			if modTime == 0 {
				modTime = posterTime
			}
			data = append(data, xnode(
				xleaf("title", cdataParse(subject)),
				xleaf("link", msgURL),
				xleaf("summary", cdataParse(body)),
				xnodeK("author", xleaf("name", posterName)),
				xleaf("created", gmAtom(posterTime)),
				xleaf("issued", gmAtom(posterTime)),
				xleaf("modified", gmAtom(modTime)),
				xleaf("id", msgURL),
			))
		default:
			starterLink := ""
			if idFirstMember != 0 {
				starterLink = scripturl + "?action=profile;u=" + itoa(idFirstMember)
			}
			posterLink := ""
			if idMember != 0 {
				posterLink = scripturl + "?action=profile;u=" + itoa(idMember)
			}
			data = append(data, xnode(
				xleaf("time", stripTags(c.timeformat(posterTime))),
				xleaf("id", itoa(idMsg)),
				xleaf("subject", cdataParse(subject)),
				xleaf("body", cdataParse(body)),
				xnodeK("starter",
					xleaf("name", cdataParse(firstPosterName)),
					xleaf("id", itoa(idFirstMember)),
					xleaf("link", starterLink)),
				xnodeK("poster",
					xleaf("name", cdataParse(posterName)),
					xleaf("id", itoa(idMember)),
					xleaf("link", posterLink)),
				xnodeK("topic",
					xleaf("subject", cdataParse(firstSubject)),
					xleaf("id", itoa(idTopic)),
					xleaf("link", scripturl+"?topic="+itoa(idTopic)+".new#new")),
				xnodeK("board",
					xleaf("name", cdataParse(bname)),
					xleaf("id", itoa(idBoard)),
					xleaf("link", scripturl+"?board="+itoa(idBoard)+".0")),
				xleaf("link", msgURL),
			))
		}
	}
	prows.Close()
	return data
}

// getXmlProfile is getXmlProfile(): a single member's profile feed.
func (c *Ctx) getXmlProfile(format string) []xmlField {
	a := c.App
	scripturl := a.ScriptURL

	u := c.GET.Int("u")
	if u == 0 {
		return nil
	}
	c.loadMemberData([]int{u})
	if !c.loadMemberContext(u) || !c.allowedTo("profile_view_any") {
		return nil
	}
	m := c.memberCtx[u]
	profileURL := scripturl + "?action=profile;u=" + itoa(m.ID)
	groupOrPost := m.Group
	if groupOrPost == "" {
		groupOrPost = m.PostGroup
	}

	switch {
	case format == "rss" || format == "rss2":
		return []xmlField{xnode(
			xleaf("title", cdataParse(m.Name)),
			xleaf("link", profileURL),
			xleaf("description", cdataParse(groupOrPost)),
			xleaf("comments", scripturl+"?action=pm;sa=send;u="+itoa(m.ID)),
			xleaf("pubDate", gmdateRSS(m.RegisteredTS)),
			xleaf("guid", profileURL),
		)}
	case format == "rdf":
		return []xmlField{xnode(
			xleaf("title", cdataParse(m.Name)),
			xleaf("link", profileURL),
			xleaf("description", cdataParse(groupOrPost)),
		)}
	case format == "atom":
		return []xmlField{xnode(
			xleaf("title", cdataParse(m.Name)),
			xleaf("link", profileURL),
			xleaf("summary", cdataParse(groupOrPost)),
			xleaf("created", gmAtom(m.RegisteredTS)),
			xleaf("issued", gmAtom(m.RegisteredTS)),
			xleaf("modified", gmAtom(m.LastLoginTS)),
			xleaf("id", profileURL),
		)}
	}

	// smf format: a flat, richly-detailed record.
	hideContacts := !a.SettingEmpty("guest_hideContacts") && c.User.IsGuest
	data := []xmlField{
		xleaf("username", cdataParse(m.Username)),
		xleaf("name", cdataParse(m.Name)),
		xleaf("link", profileURL),
		xleaf("posts", m.Posts),
		xleaf("post-group", cdataParse(m.PostGroup)),
		xleaf("language", cdataParse(m.Language)),
		xleaf("last-login", gmdateRSS(m.LastLoginTS)),
		xleaf("registered", gmdateRSS(m.RegisteredTS)),
	}
	if m.GenderName != "" {
		data = append(data, xleaf("gender", cdataParse(m.GenderName)))
	}
	if m.AvatarName != "" {
		data = append(data, xleaf("avatar", m.AvatarURL))
	}
	if m.IsOnline {
		data = append(data, xleaf("online", ""))
	}
	if m.Signature != "" {
		data = append(data, xleaf("signature", cdataParse(m.Signature)))
	}
	if m.Blurb != "" {
		data = append(data, xleaf("blurb", cdataParse(m.Blurb)))
	}
	if m.Location != "" {
		data = append(data, xleaf("location", cdataParse(m.Location)))
	}
	if m.Title != "" {
		data = append(data, xleaf("title", cdataParse(m.Title)))
	}
	if m.WebsiteTitle != "" {
		data = append(data, xnodeK("website",
			xleaf("title", cdataParse(m.WebsiteTitle)),
			xleaf("link", m.WebsiteURL)))
	}
	if m.Group != "" {
		data = append(data, xleaf("postition", cdataParse(m.Group)))
	}
	if (!m.HideEmail || a.SettingEmpty("allow_hideEmail")) && !hideContacts {
		data = append(data, xleaf("email", m.Email))
	}
	if m.BirthDate != "" && substr(m.BirthDate, 0, 4) != "0000" {
		var by, bm, bd int
		fmtScan(m.BirthDate, &by, &bm, &bd)
		now := time.Unix(c.forumTime(false, 0), 0).UTC()
		age := now.Year() - by
		if !(int(now.Month()) > bm || (int(now.Month()) == bm && now.Day() >= bd)) {
			age--
		}
		data = append(data, xleaf("age", itoa(age)))
	}
	return data
}

// fmtScan parses "YYYY-MM-DD" into year/month/day (sscanf '%d-%d-%d').
func fmtScan(s string, y, mo, d *int) {
	parts := strings.SplitN(s, "-", 3)
	if len(parts) > 0 {
		*y = atoi(parts[0])
	}
	if len(parts) > 1 {
		*mo = atoi(parts[1])
	}
	if len(parts) > 2 {
		*d = atoi(parts[2])
	}
}
