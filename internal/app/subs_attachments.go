package app

// Attachment filename resolution (getAttachmentFilename and
// getLegacyAttachmentFilename from Subs.php) and removeAttachments from
// ManageAttachments.php. The rest of ManageAttachments.php ports with
// Phase 7.

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

// getAttachmentFilename is getAttachmentFilename($filename, $attachment_id,
// $new, $file_hash).
func (a *App) getAttachmentFilename(filename string, attachmentID int, isNew bool, fileHash string) string {
	// Just make up a nice hash...
	if isNew {
		h := md5.Sum([]byte(filename + itoa(int(time.Now().Unix()))))
		return fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%x", h)+itoa(rand.Int()))))
	}

	// Grab the file hash if it wasn't added.
	if fileHash == "" {
		if err := a.DB.QueryRow(a.Q(`SELECT file_hash FROM {$db_prefix}attachments WHERE ID_ATTACH = ?`),
			attachmentID).Scan(&fileHash); err != nil {
			return ""
		}
	}

	// In case of files from the old system, do a legacy call.
	if fileHash == "" {
		return a.getLegacyAttachmentFilename(filename, attachmentID, false)
	}

	return a.Setting("attachmentUploadDir") + "/" + itoa(attachmentID) + "_" + fileHash
}

var (
	legacyFromBytes = []byte{
		0x8a, 0x8e, 0x9a, 0x9e, 0x9f, 0xc0, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc7,
		0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xce, 0xcf, 0xd1, 0xd2, 0xd3, 0xd4,
		0xd5, 0xd6, 0xd8, 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xe0, 0xe1, 0xe2, 0xe3,
		0xe4, 0xe5, 0xe7, 0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef, 0xf1,
		0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xff,
	}
	legacyToBytes = "SZszYAAAAAACEEEEIIIINOOOOOOUUUUYaaaaaaceeeeiiiinoooooouuuuyy"

	legacyMulti = map[byte]string{
		0xde: "TH", 0xfe: "th", 0xd0: "DH", 0xf0: "dh", 0xdf: "ss",
		0x8c: "OE", 0x9c: "oe", 0xc6: "AE", 0xe6: "ae", 0xb5: "u",
	}

	legacySpaceRe   = regexp.MustCompile(`\s`)
	legacyNonWordRe = regexp.MustCompile(`[^\w_\.\-]`)
	legacyDotsRe    = regexp.MustCompile(`\.[\.]+`)
)

// getLegacyAttachmentFilename is getLegacyAttachmentFilename().
func (a *App) getLegacyAttachmentFilename(filename string, attachmentID int, isNew bool) string {
	// Remove special accented characters - ie. sí. (ISO-8859-1 bytes.)
	var b strings.Builder
	for i := 0; i < len(filename); i++ {
		ch := filename[i]
		replaced := false
		for j, f := range legacyFromBytes {
			if ch == f {
				b.WriteByte(legacyToBytes[j])
				replaced = true
				break
			}
		}
		if replaced {
			continue
		}
		if multi, ok := legacyMulti[ch]; ok {
			b.WriteString(multi)
		} else {
			b.WriteByte(ch)
		}
	}
	cleanName := b.String()

	// Sorry, no spaces, dots, or anything else but letters allowed.
	cleanName = legacySpaceRe.ReplaceAllString(cleanName, "_")
	cleanName = legacyNonWordRe.ReplaceAllString(cleanName, "")

	encName := itoa(attachmentID) + "_" + strings.ReplaceAll(cleanName, ".", "_") +
		fmt.Sprintf("%x", md5.Sum([]byte(cleanName)))
	cleanName = legacyDotsRe.ReplaceAllString(cleanName, ".")

	if attachmentID == 0 || (isNew && a.SettingEmpty("attachmentEncryptFilenames")) {
		return cleanName
	} else if isNew {
		return encName
	}

	if _, err := os.Stat(a.Setting("attachmentUploadDir") + "/" + encName); err == nil {
		return a.Setting("attachmentUploadDir") + "/" + encName
	}
	return a.Setting("attachmentUploadDir") + "/" + cleanName
}

// removeAttachments is removeAttachments($condition, $query_type,
// $return_affected_messages, $autoThumbRemoval) from ManageAttachments.php.
func (a *App) removeAttachments(condition, queryType string, returnAffectedMessages, autoThumbRemoval bool) []int {
	var msgs []int
	var attach []int
	var parents []int

	msgCol := ", a.ID_MSG"
	fromExtra := ""
	whereExtra := ""
	switch queryType {
	case "messages":
		msgCol = ", m.ID_MSG"
		fromExtra = ", {$db_prefix}messages AS m"
		whereExtra = `
			AND m.ID_MSG = a.ID_MSG`
	case "members":
		fromExtra = ", {$db_prefix}members AS mem"
		whereExtra = `
			AND mem.ID_MEMBER = a.ID_MEMBER`
	}

	// Get all the attachment names and ID_MSGs.
	rows, err := a.DB.Query(a.Q(`
		SELECT
			a.filename, a.file_hash, a.attachmentType, a.ID_ATTACH, a.ID_MEMBER` + msgCol + `,
			IFNULL(thumb.ID_ATTACH, 0) AS ID_THUMB, IFNULL(thumb.filename, '') AS thumb_filename,
			IFNULL(thumb_parent.ID_ATTACH, 0) AS ID_PARENT, IFNULL(thumb.file_hash, '') AS thumb_file_hash
		FROM {$db_prefix}attachments AS a` + fromExtra + `
			LEFT JOIN {$db_prefix}attachments AS thumb ON (thumb.ID_ATTACH = a.ID_THUMB)
			LEFT JOIN {$db_prefix}attachments AS thumb_parent ON (a.attachmentType = 3 AND thumb_parent.ID_THUMB = a.ID_ATTACH)
		WHERE ` + condition + whereExtra))
	if err != nil {
		return nil
	}
	for rows.Next() {
		var filename, fileHash, thumbFilename, thumbFileHash string
		var attachmentType, idAttach, idMember, idMsg, idThumb, idParent int
		rows.Scan(&filename, &fileHash, &attachmentType, &idAttach, &idMember, &idMsg,
			&idThumb, &thumbFilename, &idParent, &thumbFileHash)

		// Figure out the "encrypted" filename and unlink it ;).
		if attachmentType == 1 {
			os.Remove(a.Setting("custom_avatar_dir") + "/" + filename)
		} else {
			if fn := a.getAttachmentFilename(filename, idAttach, false, fileHash); fn != "" {
				os.Remove(fn)
			}

			// If this was a thumb, the parent attachment should know about
			// it.
			if idParent != 0 {
				parents = append(parents, idParent)
			}

			// If this attachments has a thumb, remove it as well.
			if idThumb != 0 && autoThumbRemoval {
				if fn := a.getAttachmentFilename(thumbFilename, idThumb, false, thumbFileHash); fn != "" {
					os.Remove(fn)
				}
				attach = append(attach, idThumb)
			}
		}

		// Make a list.
		if returnAffectedMessages && attachmentType == 0 {
			msgs = append(msgs, idMsg)
		}
		attach = append(attach, idAttach)
	}
	rows.Close()

	// Removed attachments don't have to be updated anymore.
	var keepParents []string
	for _, p := range parents {
		removed := false
		for _, at := range attach {
			if at == p {
				removed = true
				break
			}
		}
		if !removed {
			keepParents = append(keepParents, itoa(p))
		}
	}
	if len(keepParents) > 0 {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}attachments
			SET ID_THUMB = 0
			WHERE ID_ATTACH IN (` + strings.Join(keepParents, ", ") + `)`))
	}

	if len(attach) > 0 {
		ids := make([]string, len(attach))
		for i, at := range attach {
			ids[i] = itoa(at)
		}
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}attachments
			WHERE ID_ATTACH IN (` + strings.Join(ids, ", ") + `)`))
	}

	if returnAffectedMessages {
		return uniqueInts(msgs)
	}
	return nil
}

// attachmentOptions mirrors createAttachment(&$attachmentOptions).
type attachmentOptions struct {
	Post     int
	Poster   int
	Name     string
	TmpName  string // either an uploaded temp path or a post_tmp_* name
	Size     int
	FileHash string

	// outputs:
	ID          int
	Thumb       int
	Width       int
	Height      int
	Destination string
	Errors      []string
}

var postTmpAnyRe = regexp.MustCompile(`^post_tmp_\d+_\d+$`)

// createAttachment is createAttachment() from Subs-Post.php.
func (c *Ctx) createAttachment(opt *attachmentOptions) bool {
	a := c.App

	opt.Errors = nil

	alreadyUploaded := regexp.MustCompile(`^post_tmp_` + itoa(opt.Poster) + `_\d+$`).MatchString(opt.TmpName)
	if alreadyUploaded {
		opt.TmpName = a.Setting("attachmentUploadDir") + "/" + opt.TmpName
	}

	// Make sure the file actually exists... sometimes it doesn't.
	if _, err := os.Stat(opt.TmpName); err != nil {
		opt.Errors = []string{"could_not_upload"}
		return false
	}

	opt.Width, opt.Height, _ = imageSize(opt.TmpName)

	// Get the hash if no hash has been given yet.
	if opt.FileHash == "" {
		opt.FileHash = a.getAttachmentFilename(opt.Name, 0, true, "")
	}

	// Is the file too big?
	if !a.SettingEmpty("attachmentSizeLimit") && opt.Size > a.SettingInt("attachmentSizeLimit")*1024 {
		opt.Errors = append(opt.Errors, "too_large")
	}

	if !a.SettingEmpty("attachmentCheckExtensions") {
		ext := ""
		if i := strings.LastIndexByte(opt.Name, '.'); i >= 0 {
			ext = strings.ToLower(opt.Name[i+1:])
		}
		ok := false
		for _, allowed := range strings.Split(strings.ToLower(a.Setting("attachmentExtensions")), ",") {
			if strings.TrimSpace(allowed) == ext {
				ok = true
				break
			}
		}
		if !ok {
			opt.Errors = append(opt.Errors, "bad_extension")
		}
	}

	if !a.SettingEmpty("attachmentDirSizeLimit") {
		// Make sure the directory isn't full.
		dirSize := 0
		entries, err := os.ReadDir(a.Setting("attachmentUploadDir"))
		if err != nil {
			c.fatalLangError("smf115b", true)
		}
		for _, e := range entries {
			if postTmpAnyRe.MatchString(e.Name()) {
				// Temp file is more than 5 hours old!
				if info, err := e.Info(); err == nil && info.ModTime().Unix() < time.Now().Unix()-18000 {
					os.Remove(a.Setting("attachmentUploadDir") + "/" + e.Name())
				}
				continue
			}
			if info, err := e.Info(); err == nil {
				dirSize += int(info.Size())
			}
		}

		// Too big!  Maybe you could zip it or something...
		if opt.Size+dirSize > a.SettingInt("attachmentDirSizeLimit")*1024 {
			opt.Errors = append(opt.Errors, "directory_full")
		}
	}

	// Check if the file already exists.... (for those who do not encrypt
	// their filenames...)
	if a.SettingEmpty("attachmentEncryptFilenames") {
		// Make sure they aren't trying to upload a nasty file.
		base := strings.ToLower(filepathBase(opt.Name))
		for _, bad := range []string{"con", "com1", "com2", "com3", "com4", "prn", "aux", "lpt1", ".htaccess", "index.php"} {
			if base == bad {
				opt.Errors = append(opt.Errors, "bad_filename")
				break
			}
		}

		// Check if there's another file with that name...
		var dummy int
		if err := a.DB.QueryRow(a.Q(`
			SELECT ID_ATTACH
			FROM {$db_prefix}attachments
			WHERE filename = ?
			LIMIT 1`), strings.ToLower(opt.Name)).Scan(&dummy); err == nil {
			opt.Errors = append(opt.Errors, "taken_filename")
		}
	}

	if len(opt.Errors) > 0 {
		return false
	}

	res, err := a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}attachments
			(ID_MSG, filename, file_hash, size, width, height)
		VALUES (?, SUBSTR(?, 1, 255), ?, ?, ?, ?)`),
		opt.Post, opt.Name, opt.FileHash, opt.Size, opt.Width, opt.Height)
	if err != nil {
		return false
	}
	id, _ := res.LastInsertId()
	opt.ID = int(id)
	if opt.ID == 0 {
		return false
	}

	opt.Destination = a.getAttachmentFilename(filepathBase(opt.Name), opt.ID, false, opt.FileHash)

	if err := os.Rename(opt.TmpName, opt.Destination); err != nil {
		c.fatalLangError("smf124", true)
	}

	// Attempt to chmod it.
	os.Chmod(opt.Destination, 0644)

	if opt.Width == 0 && opt.Height == 0 {
		return true
	}

	// Like thumbnails, do we?
	if !a.SettingEmpty("attachmentThumbnails") && !a.SettingEmpty("attachmentThumbWidth") && !a.SettingEmpty("attachmentThumbHeight") &&
		(opt.Width > a.SettingInt("attachmentThumbWidth") || opt.Height > a.SettingInt("attachmentThumbHeight")) {
		if a.createThumbnail(opt.Destination, a.SettingInt("attachmentThumbWidth"), a.SettingInt("attachmentThumbHeight")) {
			// Figure out how big we actually made it.
			thumbWidth, thumbHeight, _ := imageSize(opt.Destination + "_thumb")
			thumbFilename := opt.Name + "_thumb"
			thumbSize := 0
			if info, err := os.Stat(opt.Destination + "_thumb"); err == nil {
				thumbSize = int(info.Size())
			}

			// To the database we go!
			thumbFileHash := a.getAttachmentFilename(thumbFilename, 0, true, "")
			res, err := a.DB.Exec(a.Q(`
				INSERT INTO {$db_prefix}attachments
					(ID_MSG, attachmentType, filename, file_hash, size, width, height)
				VALUES (?, 3, SUBSTR(?, 1, 255), ?, ?, ?, ?)`),
				opt.Post, thumbFilename, thumbFileHash, thumbSize, thumbWidth, thumbHeight)
			if err == nil {
				tid, _ := res.LastInsertId()
				opt.Thumb = int(tid)

				if opt.Thumb != 0 {
					a.DB.Exec(a.Q(`
						UPDATE {$db_prefix}attachments
						SET ID_THUMB = ?
						WHERE ID_ATTACH = ?`), opt.Thumb, opt.ID)

					os.Rename(opt.Destination+"_thumb",
						a.getAttachmentFilename(thumbFilename, opt.Thumb, false, thumbFileHash))
				}
			}
		}
	}

	return true
}

// filepathBase is PHP basename() (works on both / and \).
func filepathBase(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		return name[i+1:]
	}
	return name
}
