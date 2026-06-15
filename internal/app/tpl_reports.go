package app

// Ports of Themes/default/Reports.template.php: template_report_type,
// template_main, and the print page (template_print_above/print/print_below,
// wired through the reports_print layer).

// templateReportType is template_report_type().
func templateReportType(c *Ctx) {
	page := c.Page.(*reportsPage)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=reports" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" cellspacing="0" cellpadding="4" width="100%" class="tborder">
				<tr class="titlebg">
					<td colspan="2">`, c.Txt("generate_reports"), `</td>
				</tr>
				<tr class="windowbg">
					<td class="smalltext" style="padding: 2ex;" colspan="2">`, c.Txt("generate_reports_desc"), `</td>
				</tr>
				<tr class="titlebg">
					<td colspan="2">`, c.Txt("generate_reports_type"), `:</td>
				</tr>`)
	alternate := false
	for _, t := range page.ReportTypes {
		rowClass := "windowbg2"
		if alternate {
			rowClass = "windowbg"
		}
		c.O(`
				<tr class="`, rowClass, `" valign="top">
					<td width="20">
						<input type="radio" id="rt_`, t.ID, `" name="rt" value="`, t.ID, `"`, checkedIf(t.IsFirst), ` class="check" />
					</td>
					<td align="left" width="100%">
						<label for="rt_`, t.ID, `">
							<b>`, t.Title, `</b>`)
		if t.Description != "" {
			c.O(`
							<br /><span class="smalltext">`, t.Description, `</span>`)
		}
		c.O(`
						</label>
					</td>
				</tr>`)
		alternate = !alternate
	}
	c.O(`
				<tr class="titlebg">
					<td align="right" colspan="2">
						<input type="submit" name="continue" value="`, c.Txt("generate_reports_continue"), `" />
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// reportTableRows renders the shared <tr>/<td> body for one table (used by both
// the on-screen and print templates, which differ only in indentation/wrapper).
func reportTableRows(c *Ctx, t *rptTable) {
	row_number := 0
	alternate := false
	for _, row := range t.Data {
		if row_number == 0 && t.ShadeTop {
			c.O(`
			<tr class="titlebg" valign="top">`)
		} else {
			rowClass := "windowbg2"
			if alternate {
				rowClass = "windowbg"
			}
			c.O(`
			<tr class="`, rowClass, `" valign="top">`)
		}
		column_number := 0
		for _, data := range row {
			if data.Seperator && column_number == 0 {
				c.O(`
				<td colspan="`, t.ColumnCount, `" class="catbg">
					<b>`, data.Value, `:</b>
				</td>`)
				break
			}
			if column_number == 0 && t.ShadeLeft {
				widthAttr := ""
				if t.WidthShaded != "auto" {
					widthAttr = `width="` + t.WidthShaded + `"`
				}
				shadedVal := ""
				if data.Value != t.DefaultValue {
					shadedVal = data.Value
					if !empty(data.Value) {
						shadedVal += ":"
					}
				}
				c.O(`
				<td align="`, t.AlignShaded, `" class="titlebg" `, widthAttr, `>
					`, shadedVal, `
				</td>`)
			} else {
				widthAttr := ""
				if t.WidthNormal != "auto" {
					widthAttr = `width="` + t.WidthNormal + `"`
				}
				styleAttr := ""
				if data.Style != "" {
					styleAttr = `style="` + data.Style + `"`
				}
				c.O(`
				<td align="`, t.AlignNormal, `" `, widthAttr, ` `, styleAttr, `>
					`, data.Value, `
				</td>`)
			}
			column_number++
		}
		c.O(`
			</tr>`)
		row_number++
		alternate = !alternate
	}
}

// templateReportsMain is template_main().
func templateReportsMain(c *Ctx) {
	b := c.Page.(*reportBuilder)
	scripturl := c.App.ScriptURL

	c.O(`
		<div class="tborder">
			<div class="titlebg" style="padding: 4px;">
				<div style="float: left;"><b>`, c.Txt("results"), `</b></div>
				<div style="text-align: right;">&nbsp;`)
	if c.Theme.Empty("use_tabs") {
		printLabel := c.Txt("465")
		if !c.Theme.Empty("use_image_buttons") {
			printLabel = `<img src="` + c.Theme.ImagesURL() + `/` + c.User.Language + `/print.gif" alt="` + c.Txt("465") + `" border="0" />`
		}
		c.O(`
					<a href="`, scripturl, `?action=reports;rt=`, c.reportType, `;st=print" target="_blank">`, printLabel, `</a>`)
	}
	c.O(`
				</div>
			</div>
		</div>`)
	if !c.Theme.Empty("use_tabs") {
		c.O(`
		<table width="100%" cellpadding="0" cellspacing="0" border="0"><tr>
			<td align="right">
				<table cellpadding="0" cellspacing="0" border="0">
					<tr>
						<td class="maintab_first">&nbsp;</td>
						<td valign="top" class="maintab_back">
							<a href="`, scripturl, `?action=reports;rt=`, c.reportType, `;st=print" target="_blank">`, c.Txt("465"), `</a>
						</td>
						<td class="maintab_last">&nbsp;</td>
					</tr>
				</table>
			</td>
		</tr></table><br />`)
	}

	for _, t := range b.Tables {
		c.O(`
		<table border="0" cellspacing="1" cellpadding="3" width="100%" class="bordercolor">`)
		if t.Title != "" {
			c.O(`
			<tr class="catbg">
				<td colspan="`, t.ColumnCount, `">`, t.Title, `</td>
			</tr>`)
		}
		reportTableRows(c, t)
		c.O(`
		</table>
		<br />`)
	}
}

// templateReportsPrint is template_print().
func templateReportsPrint(c *Ctx) {
	b := c.Page.(*reportBuilder)
	for _, t := range b.Tables {
		widthStyle := ""
		if t.MaxWidth != "auto" {
			widthStyle = "width: " + t.MaxWidth + "px;"
		}
		c.O(`
		<div style="overflow: visible; `, widthStyle, `">
			<table border="0" cellspacing="1" cellpadding="4" width="100%" class="bordercolor">`)
		if t.Title != "" {
			c.O(`
				<tr class="catbg">
					<td colspan="`, t.ColumnCount, `">
						`, t.Title, `
					</td>
				</tr>`)
		}
		reportTableRows(c, t)
		c.O(`
			</table>
		</div><br />`)
	}
}

// templateReportsPrintAbove is template_print_above().
func templateReportsPrintAbove(c *Ctx) {
	c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"`)
	if c.RightToLeft {
		c.O(` dir="rtl"`)
	}
	c.O(`>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=`, c.CharacterSet, `" />
		<title>`, c.PageTitle, `</title>
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
			.catbg
			{
				background-color: #D6D6D6;
				font-weight: bold;
			}
			.titlebg, tr.titlebg td, .titlebg a:link, .titlebg a:visited
			{
				font-style: normal;
				background-color: #F5EDED;
			}
			.bordercolor
			{
				background-color: #333;
			}
			.windowbg
			{
				color: black;
				background-color: white;
			}
			.windowbg2
			{
				color: black;
				background-color: #F1F1F1;
			}
		</style>`)
	if c.Browser.NeedsSizeFix {
		c.O(`
		<link rel="stylesheet" type="text/css" href="`, c.Theme.DefaultThemeURL(), `/fonts-compat.css" />`)
	}
	c.O(`
	</head>
	<body>`)
}

// templateReportsPrintBelow is template_print_below().
func templateReportsPrintBelow(c *Ctx) {
	c.O(`
		<div align="center" style="margin-top: 2ex;" class="smalltext">`)
	c.themeCopyright()
	c.O(`</div>
	</body>
</html>`)
}
