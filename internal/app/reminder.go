package app

// Port of Sources/Reminder.php.

import (
	"crypto/subtle"
	"strings"
)

func init() {
	registerAction("reminder", (*Ctx).RemindMe)
}

// ReminderCtx is the page context for the Reminder templates.
type ReminderCtx struct {
	Description    string
	Code           string
	MemID          int
	RemindUser     string
	SecretQuestion string
}

// RemindMe is RemindMe(): the controlling delegator.
func (c *Ctx) RemindMe() {
	c.loadLanguage("Profile")
	c.PageTitle = c.App.Config.MbName + " " + c.Txt("669")
	c.Page = &ReminderCtx{}
	c.SubTemplate = templateReminderMain

	switch c.REQUEST.Str("sa") {
	case "mail":
		c.RemindMail()
	case "secret":
		c.secretAnswerInput()
	case "secret2":
		c.secretAnswer2()
	case "setpassword":
		c.setPassword()
	case "setpassword2":
		c.setPassword2()
	}
}

// RemindMail is RemindMail(): email a reminder.
func (c *Ctx) RemindMail() {
	a := c.App

	c.checkSession("post", "", true)

	// You must enter a username/email address.
	if c.POST.Str("user") == "" {
		c.fatalLangError("40", false)
	}

	var idMember, isActivated int
	var realName, memberName, emailAddress, validationCode string
	scan := func(where string) error {
		return a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER, realName, memberName, emailAddress, is_activated, validation_code
			FROM {$db_prefix}members
			WHERE `+where+` = ?
			LIMIT 1`), c.POST.Str("user")).Scan(&idMember, &realName, &memberName, &emailAddress, &isActivated, &validationCode)
	}
	if scan("memberName") != nil {
		if scan("emailAddress") != nil {
			c.fatalLangError("40", false)
		}
	}

	// If the user isn't activated/approved, give them some feedback.
	if isActivated != 1 {
		key := "registration_not_activated"
		if strings.TrimSpace(validationCode) == "" {
			key = "registration_not_approved"
		}
		c.fatalError(c.Txt(key)+` <a href="`+a.ScriptURL+`?action=activate;user=`+Htmlspecialchars(c.POST.Str("user"))+`">`+c.Txt("662")+`</a>.`, false)
	}

	// You can't get emailed if you have no email address.
	emailAddress = strings.TrimSpace(emailAddress)
	if emailAddress == "" {
		c.fatalError(`<b>`+c.Txt("394")+`<br />`+c.Txt("395")+` <a href="mailto:`+a.Config.WebmasterEmail+`">webmaster</a> `+c.Txt("396")+`.`, true)
	}

	// Randomly generate a new password.
	password := generateValidationCode()

	// Set the password in the database. Store the full md5 of the (now
	// full-length) token rather than a 10-char prefix, so the stored verifier
	// can't be brute-forced (a deliberate divergence from SMF's truncation).
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET validation_code = ? WHERE ID_MEMBER = ?`),
		md5hex(password), idMember)

	c.sendmail([]string{emailAddress}, c.Txt("reminder_subject"),
		phpSprintf(c.Txt("sendtopic_dear"), realName)+"\n\n"+
			c.Txt("reminder_mail")+":\n\n"+
			a.ScriptURL+"?action=reminder;sa=setpassword;u="+itoa(idMember)+";code="+password+"\n\n"+
			c.Txt("512")+": "+c.User.IP+"\n\n"+
			c.Txt("35")+": "+memberName+"\n\n"+
			c.Txt("130"), "")

	// Set up the template.
	c.PageTitle = c.Txt("194")
	c.Page = &ReminderCtx{Description: c.Txt("reminder_sent")}
	c.SubTemplate = templateReminderSent
}

// setPassword is setPassword(): show the new password form.
func (c *Ctx) setPassword() {
	if !c.REQUEST.Has("code") {
		c.fatalLangError("1", true)
	}

	c.PageTitle = c.Txt("reminder_set_password")
	c.Page = &ReminderCtx{
		Code:  c.REQUEST.Str("code"),
		MemID: c.REQUEST.Int("u"),
	}
	c.SubTemplate = templateReminderSetPassword
}

// setPassword2 is setPassword2(): apply the new password.
func (c *Ctx) setPassword2() {
	a := c.App

	if empty(c.POST.Str("u")) || !c.POST.Has("passwrd1") || !c.POST.Has("passwrd2") {
		c.fatalLangError("1", false)
	}

	memID := c.POST.Int("u")

	if c.POST.Str("passwrd1") != c.POST.Str("passwrd2") {
		c.fatalLangError("213", false)
	}
	if c.POST.Str("passwrd1") == "" {
		c.fatalLangError("91", false)
	}

	c.loadLanguage("Login")

	// Get the code as it should be from the database.
	var realCode, username, email string
	err := a.DB.QueryRow(a.Q(`
		SELECT validation_code, memberName, emailAddress
		FROM {$db_prefix}members
		WHERE ID_MEMBER = ?
			AND is_activated = 1
			AND validation_code != ''
		LIMIT 1`), memID).Scan(&realCode, &username, &email)
	if err != nil {
		c.fatalLangError("invalid_userid", false)
	}

	// Is the password actually valid?
	if pwErr := c.validatePassword(c.POST.Str("passwrd1"), username, []string{email}); pwErr != "" {
		c.fatalLangError("profile_error_password_"+pwErr, false)
	}

	// Quit if this code is not right. Compare the full hash in constant time
	// (SMF compared only a 10-char prefix with ==).
	if empty(c.POST.Str("code")) ||
		subtle.ConstantTimeCompare([]byte(realCode), []byte(md5hex(c.POST.Str("code")))) != 1 {
		c.fatalError(c.Txt("invalid_activation_code"), false)
	}

	// User validated.  Update the database!
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET validation_code = '', passwd = ? WHERE ID_MEMBER = ?`),
		sha1hex(strings.ToLower(username)+c.POST.Str("passwrd1")), memID)

	c.PageTitle = c.Txt("reminder_password_set")
	c.Page = &LoginCtx{
		DefaultUsername: username,
		DefaultPassword: c.POST.Str("passwrd1"),
		Description:     c.Txt("reminder_password_set"),
	}
	c.SubTemplate = templateLogin
}

// secretAnswerInput is secretAnswerInput(): ask the secret question.
func (c *Ctx) secretAnswerInput() {
	a := c.App

	c.checkSession("post", "", true)

	// Please provide an email or user....
	if c.POST.Str("user") == "" {
		c.fatalLangError("40", false)
	}

	var realName, memberName, secretQuestion string
	scan := func(where string) error {
		return a.DB.QueryRow(a.Q(`
			SELECT realName, memberName, secretQuestion
			FROM {$db_prefix}members
			WHERE `+where+` = ?
			LIMIT 1`), c.POST.Str("user")).Scan(&realName, &memberName, &secretQuestion)
	}
	if scan("memberName") != nil {
		if scan("emailAddress") != nil {
			c.fatalLangError("40", false)
		}
	}

	// If there is NO secret question - then throw an error.
	if strings.TrimSpace(secretQuestion) == "" {
		c.fatalLangError("registration_no_secret_question", false)
	}

	// Ask for the answer...
	c.Page = &ReminderCtx{
		RemindUser:     memberName,
		SecretQuestion: secretQuestion,
	}
	c.SubTemplate = templateReminderAsk
}

// secretAnswer2 is secretAnswer2(): check the answer, set the password.
func (c *Ctx) secretAnswer2() {
	a := c.App

	c.checkSession("post", "", true)

	// Hacker?  How did you get this far without an email or username?
	if c.POST.Str("user") == "" {
		c.fatalLangError("40", false)
	}

	c.loadLanguage("Login")

	var idMember int
	var realName, memberName, secretAnswer, secretQuestion string
	scan := func(where string) error {
		return a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER, realName, memberName, secretAnswer, secretQuestion
			FROM {$db_prefix}members
			WHERE `+where+` = ?
			LIMIT 1`), c.POST.Str("user")).Scan(&idMember, &realName, &memberName, &secretAnswer, &secretQuestion)
	}
	if scan("memberName") != nil {
		if scan("emailAddress") != nil {
			c.fatalLangError("40", false)
		}
	}

	// Check if the secret answer is correct.
	if secretQuestion == "" || secretAnswer == "" || md5hex(c.POST.Str("secretAnswer")) != secretAnswer {
		c.logError(phpSprintf(c.Txt("reminder_error"), memberName))
		c.fatalLangError("pswd7", false)
	}

	// You can't use a blank one!
	if strings.TrimSpace(c.POST.Str("passwrd1")) == "" {
		c.fatalLangError("38", false)
	}

	// They have to be the same too.
	if c.POST.Str("passwrd1") != c.POST.Str("passwrd2") {
		c.fatalLangError("213", false)
	}

	// Alright, so long as 'yer sure.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET passwd = ? WHERE ID_MEMBER = ?`),
		sha1hex(strings.ToLower(memberName)+c.POST.Str("passwrd1")), idMember)

	// Tell them it went fine.
	c.PageTitle = c.Txt("reminder_password_set")
	c.Page = &LoginCtx{
		DefaultUsername: memberName,
		DefaultPassword: c.POST.Str("passwrd1"),
		Description:     c.Txt("reminder_password_set"),
	}
	c.SubTemplate = templateLogin
}

// ---- Reminder templates (Themes/default/Reminder.template.php) ----

// templateReminderMain is template_main().
func templateReminderMain(c *Ctx) {
	scripturl := c.App.ScriptURL
	c.O(`
	<br />
	<form action="`, scripturl, `?action=reminder;sa=mail" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" width="400" cellspacing="0" cellpadding="4" align="center" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("194"), `</td>
			</tr><tr class="windowbg">
				<td colspan="2" class="smalltext" style="padding: 2ex;">`, c.Txt("pswd4"), `</td>
			</tr><tr class="windowbg2">
				<td width="40%">`, c.Txt("smf100"), `:</td>
				<td><input type="text" name="user" size="30" /></td>
			</tr><tr class="windowbg2">
				<td colspan="2" align="center"><label for="secret"><input type="checkbox" name="sa" value="secret" id="secret" class="check" /> `, c.Txt("pswd3"), `.</label></td>
			</tr><tr class="windowbg2">
				<td colspan="2" align="center"><input type="submit" value="`, c.Txt("sendtopic_send"), `" /></td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateReminderSent is template_sent().
func templateReminderSent(c *Ctx) {
	page := c.Page.(*ReminderCtx)
	c.O(`
		<br />
		<table border="0" width="80%" cellspacing="0" cellpadding="4" class="tborder" align="center">
			<tr class="titlebg">
				<td>`, c.PageTitle, `</td>
			</tr><tr>
				<td class="windowbg" align="left" cellpadding="3" style="padding-top: 3ex; padding-bottom: 3ex;">
					`, page.Description, `
				</td>
			</tr>
		</table>`)
}

// templateReminderSetPassword is template_set_password().
func templateReminderSetPassword(c *Ctx) {
	page := c.Page.(*ReminderCtx)
	scripturl := c.App.ScriptURL
	c.O(`
	<br />
	<form action="`, scripturl, `?action=reminder;sa=setpassword2" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" width="440" cellspacing="0" cellpadding="4" class="tborder" align="center">
			<tr class="titlebg">
				<td colspan="2">`, c.PageTitle, `</td>
			</tr><tr class="windowbg">
				<td width="45%">
					<b>`, c.Txt("81"), `: </b><br />
					<span class="smalltext">`, c.Txt("596"), `</span>
				</td>
				<td valign="top"><input type="password" name="passwrd1" size="22" /></td>
			</tr><tr class="windowbg">
				<td width="45%"><b>`, c.Txt("82"), `: </b></td>
				<td><input type="password" name="passwrd2" size="22" /></td>
			</tr><tr class="windowbg">
				<td colspan="2" align="right"><input type="submit" value="`, c.Txt("10"), `" /></td>
			</tr>
		</table>
		<input type="hidden" name="code" value="`, page.Code, `" />
		<input type="hidden" name="u" value="`, page.MemID, `" />
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateReminderAsk is template_ask().
func templateReminderAsk(c *Ctx) {
	page := c.Page.(*ReminderCtx)
	scripturl := c.App.ScriptURL
	c.O(`
	<br />
	<form action="`, scripturl, `?action=reminder;sa=secret2" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" width="440" cellspacing="0" cellpadding="4" class="tborder" align="center">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("194"), `</td>
			</tr><tr class="windowbg">
				<td colspan="2" class="smalltext" style="padding: 2ex;">`, c.Txt("pswd6"), `</td>
			</tr><tr class="windowbg2">
				<td width="45%"><b>`, c.Txt("pswd1"), `:</b></td>
				<td>`, page.SecretQuestion, `</td>
			</tr><tr class="windowbg2">
				<td width="45%"><b>`, c.Txt("pswd2"), `:</b> </td>
				<td><input type="text" name="secretAnswer" size="22" /></td>
			</tr><tr class="windowbg2">
				<td width="45%">
					<b>`, c.Txt("81"), `: </b><br />
					<span class="smalltext">`, c.Txt("596"), `</span>
				</td>
				<td valign="top"><input type="password" name="passwrd1" size="22" /></td>
			</tr><tr class="windowbg2">
				<td width="45%"><b>`, c.Txt("82"), `: </b></td>
				<td><input type="password" name="passwrd2" size="22" /></td>
			</tr><tr class="windowbg2">
				<td colspan="2" align="right" style="padding: 1ex;"><input type="submit" value="`, c.Txt("10"), `" /></td>
			</tr>
		</table>

		<input type="hidden" name="user" value="`, page.RemindUser, `" />
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
