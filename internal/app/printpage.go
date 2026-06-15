package app

// Port of Sources/Printpage.php (PrintTopic) and
// Themes/default/Printpage.template.php.

func init() {
	registerAction("printpage", (*Ctx).PrintTopic)
	layerFuncs["print_above"] = templatePrintAbove
	layerFuncs["print_below"] = templatePrintBelow
}

// PrintPost is one printable post.
type PrintPost struct {
	Subject string
	Member  string
	Time    string
	Body    string
}

// PrintCtx is the page context for the Printpage template.
type PrintCtx struct {
	BoardName    string
	CategoryName string
	PosterName   string
	PostTime     string
	TopicSubject string
	Posts        []PrintPost
}

// PrintTopic is PrintTopic(): format a topic to be printer friendly.
func (c *Ctx) PrintTopic() {
	a := c.App

	if c.Topic == 0 {
		c.fatalLangError("472", false)
	}

	// Get the topic starter information.
	var posterTime int64
	var posterName string
	err := a.DB.QueryRow(a.Q(`
		SELECT m.posterTime, IFNULL(mem.realName, m.posterName) AS posterName
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE m.ID_TOPIC = ?
		ORDER BY ID_MSG
		LIMIT 1`), c.Topic).Scan(&posterTime, &posterName)
	if err != nil {
		c.fatalLangError("smf232", true)
	}

	// Lets "output" all that info.
	page := &PrintCtx{}
	c.Page = page
	c.TemplateLayers = []string{"print"}
	if c.BoardInfo != nil {
		page.BoardName = c.BoardInfo.Name
		page.CategoryName = c.BoardInfo.CatName
	}
	page.PosterName = posterName
	page.PostTime = c.timeformatNoToday(posterTime)

	// Split the topics up so we can print them.
	rows, err := a.DB.Query(a.Q(`
		SELECT subject, posterTime, body, IFNULL(mem.realName, posterName) AS posterName
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE ID_TOPIC = ?
		ORDER BY ID_MSG`), c.Topic)
	if err == nil {
		for rows.Next() {
			var subject, body, name string
			var pTime int64
			rows.Scan(&subject, &pTime, &body, &name)

			// Censor the subject and message.
			subject = c.censorText(subject)
			body = c.censorText(body)

			page.Posts = append(page.Posts, PrintPost{
				Subject: subject,
				Member:  name,
				Time:    c.timeformatNoToday(pTime),
				Body:    c.parseBBCPrint(body),
			})

			if page.TopicSubject == "" {
				page.TopicSubject = subject
			}
		}
		rows.Close()
	}

	c.SubTemplate = templatePrintMain
}

// templatePrintAbove is template_print_above().
func templatePrintAbove(c *Ctx) {
	page := c.Page.(*PrintCtx)

	rtl := ""
	if c.RightToLeft {
		rtl = ` dir="rtl"`
	}
	c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"`, rtl, `>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=`, c.CharacterSet, `" />
		<title>`, c.Txt("668"), ` - `, page.TopicSubject, `</title>
		<style type="text/css">
			body
			{
				color: black;
				background-color: white;
			}
			body, td, .normaltext
			{
				font-family: Verdana, arial, helvetica, serif;
				font-size: small;
			}
			*, a:link, a:visited, a:hover, a:active
			{
				color: black !important;
			}
			table
			{
				empty-cells: show;
			}
			.code
			{
				font-size: x-small;
				font-family: monospace;
				border: 1px solid black;
				margin: 1px;
				padding: 1px;
			}
			.quote
			{
				font-size: x-small;
				border: 1px solid black;
				margin: 1px;
				padding: 1px;
			}
			.smalltext, .quoteheader, .codeheader
			{
				font-size: x-small;
			}
			.largetext
			{
				font-size: large;
			}
			hr
			{
				height: 1px;
				border: 0;
				color: black;
				background-color: black;
			}
		</style>`)

	if c.Browser.NeedsSizeFix {
		c.O(`
		<link rel="stylesheet" type="text/css" href="`, c.Theme.DefaultThemeURL(), `/fonts-compat.css" />`)
	}

	c.O(`
	</head>
	<body>
		<h1 class="largetext">`, c.App.Config.MbName, `</h1>
		<h2 class="normaltext">`, page.CategoryName, ` => `, page.BoardName, ` => `, c.Txt("195"), `: `, page.PosterName, ` `, c.Txt("176"), ` `, page.PostTime+`</h2>

		<table width="90%" cellpadding="0" cellspacing="0" border="0">
			<tr>
				<td>`)
}

// templatePrintMain is template_main() from Printpage.template.php.
func templatePrintMain(c *Ctx) {
	page := c.Page.(*PrintCtx)

	for _, post := range page.Posts {
		c.O(`
					<br />
					<hr size="2" width="100%" />
					`, c.Txt("196"), `: <b>`, post.Subject, `</b><br />
					`, c.Txt("197"), `: <b>`, post.Member, `</b> `, c.Txt("176"), ` <b>`, post.Time, `</b>
					<hr />
					<div style="margin: 0 5ex;">`, post.Body, `</div>`)
	}
}

// templatePrintBelow is template_print_below().
func templatePrintBelow(c *Ctx) {
	c.O(`
					<br /><br />
					<div align="center" class="smalltext">`)
	c.themeCopyright()
	c.O(`</div>
				</td>
			</tr>
		</table>
	</body>
</html>`)
}
