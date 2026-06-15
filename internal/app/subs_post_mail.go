package app

// sendmail() and adminNotify() from Sources/Subs-Post.php. The Go port
// sends plain-text mail via SMTP (mail_type=1 + smtp_host) or the local
// sendmail binary, matching PHP's switch. HTML/mime alternative bodies and
// the hotmail fix are structurally ported; exact mime-encoding parity is a
// Phase 8 item (emails aren't part of the HTML golden diff).

import (
	"crypto/md5"
	"fmt"
	"net/smtp"
	"os/exec"
	"strings"
	"time"
)

// sendmail is sendmail($to, $subject, $message, $from).
func (c *Ctx) sendmail(to []string, subject, message string, from string) bool {
	a := c.App

	// Use sendmail if it's set or if no SMTP server is set.
	useSendmail := a.SettingEmpty("mail_type") || a.Setting("smtp_host") == ""

	lineBreak := "\n"
	if !useSendmail {
		lineBreak = "\r\n"
	}

	// Get rid of entities.
	subject = unHtmlspecialchars(subject)
	// Make the message use the proper line breaks.
	message = strings.ReplaceAll(strings.ReplaceAll(message, "\r", ""), "\n", lineBreak)

	fromName := a.Config.MbName
	if from != "" {
		fromName = from
	}
	mailFrom := a.Setting("mail_from")
	if mailFrom == "" {
		mailFrom = a.Config.WebmasterEmail
	}

	// Construct the mail headers...
	headers := `From: "` + fromName + `" <` + mailFrom + `>` + lineBreak
	if from != "" {
		headers += "Reply-To: <" + from + ">" + lineBreak
	}
	headers += "Return-Path: " + mailFrom + lineBreak
	headers += "Date: " + time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05") + " -0000" + lineBreak
	headers += "X-Mailer: SMF" + lineBreak
	headers += "Content-Type: text/plain; charset=" + c.CharacterSet + lineBreak
	headers += "Content-Transfer-Encoding: 7bit" + lineBreak

	ok := true
	for _, addr := range to {
		var err error
		if useSendmail {
			err = sendViaSendmail(addr, subject, message, headers, lineBreak)
		} else {
			err = sendViaSMTP(a, addr, mailFrom, subject, message, headers, lineBreak)
		}
		if err != nil {
			c.logError("sendmail: " + err.Error())
			ok = false
		}
	}
	return ok
}

func sendViaSendmail(to, subject, message, headers, lineBreak string) error {
	full := "To: " + to + lineBreak + "Subject: " + subject + lineBreak + headers + lineBreak + message + lineBreak
	cmd := exec.Command("/usr/sbin/sendmail", "-t", "-i")
	cmd.Stdin = strings.NewReader(full)
	return cmd.Run()
}

func sendViaSMTP(a *App, to, mailFrom, subject, message, headers, lineBreak string) error {
	host := a.Setting("smtp_host")
	port := a.Setting("smtp_port")
	if port == "" {
		port = "25"
	}
	addr := host + ":" + port

	var auth smtp.Auth
	if a.Setting("smtp_username") != "" {
		auth = smtp.PlainAuth("", a.Setting("smtp_username"), a.Setting("smtp_password"), host)
	}

	full := "To: " + to + lineBreak + "Subject: " + subject + lineBreak + headers + lineBreak + message + lineBreak
	return smtp.SendMail(addr, auth, mailFrom, []string{to}, []byte(full))
}

// adminNotify is adminNotify($type, $memberID, $memberName).
func (c *Ctx) adminNotify(notifyType string, memberID int, memberName string) {
	a := c.App

	// If the setting isn't enabled then just exit.
	if a.SettingEmpty("notify_new_registration") {
		return
	}

	if memberName == "" {
		a.DB.QueryRow(a.Q(`SELECT realName FROM {$db_prefix}members WHERE ID_MEMBER = ? LIMIT 1`), memberID).Scan(&memberName)
	}

	// All membergroups who can approve members.
	groups := []string{"1"}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_GROUP
		FROM {$db_prefix}permissions
		WHERE permission = 'moderate_forum'
			AND addDeny = 1
			AND ID_GROUP != 0`))
	if err == nil {
		for rows.Next() {
			var g int
			rows.Scan(&g)
			if g != 1 {
				groups = append(groups, itoa(g))
			}
		}
		rows.Close()
	}

	findInSets := "FIND_IN_SET(" + strings.Join(groups, ", additionalGroups) OR FIND_IN_SET(") + ", additionalGroups)"
	mrows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, lngfile, emailAddress
		FROM {$db_prefix}members
		WHERE (ID_GROUP IN (` + strings.Join(groups, ", ") + `) OR ` + findInSets + `)
			AND notifyTypes != 4
		ORDER BY lngfile`))
	if err != nil {
		return
	}
	defer mrows.Close()

	for mrows.Next() {
		var id int
		var lngfile, email string
		mrows.Scan(&id, &lngfile, &email)

		// Construct the message based on what they are being told.
		message := phpSprintf(c.Txt("admin_notify_profile"), memberName) + "\n\n" +
			a.ScriptURL + "?action=profile;u=" + itoa(memberID) + "\n\n"

		// If they need to be approved add more info...
		if notifyType == "approval" {
			message += c.Txt("admin_notify_approval") + "\n\n" +
				a.ScriptURL + "?action=viewmembers;sa=browse;type=approve\n\n"
		}

		// And do the actual sending...
		c.sendmail([]string{email}, c.Txt("admin_notify_subject"), message+c.Txt("130"), "")
	}
}

// md5hex is md5() in hex.
func md5hex(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
