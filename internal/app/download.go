package app

// Download() from Sources/Display.php (?action=dlattach): serve an
// attachment or avatar with permission checks.

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func init() {
	registerAction("dlattach", (*Ctx).Download)
}

// Download is Download().
func (c *Ctx) Download() {
	a := c.App

	// Make sure some attachment was requested!
	if !c.REQUEST.Has("attach") && !c.REQUEST.Has("id") {
		c.fatalLangError("1", false)
	}

	attachID := c.REQUEST.Int("attach")
	if !c.REQUEST.Has("attach") {
		attachID = c.REQUEST.Int("id")
	}

	showImage := false
	var realFilename, fileHash string
	var idAttach, attachmentType int
	var err error
	if c.REQUEST.Str("type") == "avatar" {
		err = a.DB.QueryRow(a.Q(`
			SELECT filename, ID_ATTACH, attachmentType, file_hash
			FROM {$db_prefix}attachments
			WHERE ID_ATTACH = ?
				AND ID_MEMBER > 0
			LIMIT 1`), attachID).Scan(&realFilename, &idAttach, &attachmentType, &fileHash)
		showImage = true
	} else {
		// This is just a regular attachment... This checks only the current
		// board for $board/$topic's permissions.
		c.isAllowedTo("view_attachments")

		// Make sure this attachment is on this board.
		err = a.DB.QueryRow(a.Q(`
			SELECT a.filename, a.ID_ATTACH, a.attachmentType, a.file_hash
			FROM {$db_prefix}boards AS b, {$db_prefix}messages AS m, {$db_prefix}attachments AS a
			WHERE b.ID_BOARD = m.ID_BOARD
				AND `+c.User.QuerySeeBoard+`
				AND m.ID_MSG = a.ID_MSG
				AND m.ID_TOPIC = ?
				AND a.ID_ATTACH = ?
			LIMIT 1`), c.Topic, attachID).Scan(&realFilename, &idAttach, &attachmentType, &fileHash)
		showImage = c.REQUEST.Has("image")
	}
	if err != nil {
		c.fatalLangError("1", false)
	}

	// Update the download counter (unless it's a thumbnail).
	if attachmentType != 3 {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}attachments
			SET downloads = downloads + 1
			WHERE ID_ATTACH = ?`), idAttach)
	}

	filename := a.getAttachmentFilename(realFilename, attachID, false, fileHash)

	// No point in a nicer message, because this is supposed to be an
	// attachment anyway...
	info, statErr := os.Stat(filename)
	if statErr != nil {
		c.loadLanguage("Errors")

		c.W.Header().Set("Content-Type", "text/plain; charset="+c.CharacterSet)
		c.W.WriteHeader(404)
		c.W.Write([]byte("404 - " + c.Txt("attachment_not_found")))
		c.exit()
		return
	}

	// If it hasn't been modified since the last time this attachement was
	// retrieved, there's no need to display it again.
	if ims := c.R.Header.Get("If-Modified-Since"); ims != "" {
		modifiedSince := strings.SplitN(ims, ";", 2)[0]
		if t, err := http.ParseTime(modifiedSince); err == nil && t.Unix() >= info.ModTime().Unix() {
			c.W.WriteHeader(304)
			c.exit()
			return
		}
	}

	// Check whether the ETag was sent back, and cache based on that...
	data, readErr := os.ReadFile(filename)
	if readErr != nil {
		c.fatalLangError("1", false)
	}
	fileMd5 := `"` + fmt.Sprintf("%x", md5.Sum(data)) + `"`
	if inm := c.R.Header.Get("If-None-Match"); inm != "" && strings.Contains(inm, fileMd5) {
		c.W.WriteHeader(304)
		c.exit()
		return
	}

	// Send the attachment headers.
	h := c.W.Header()
	if !c.Browser.IsGecko {
		h.Set("Content-Transfer-Encoding", "binary")
	}
	h.Set("Expires", time.Now().Add(525600*time.Minute).UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	h.Set("Last-Modified", info.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	h.Set("Accept-Ranges", "bytes")
	h.Set("Connection", "close")
	h.Set("ETag", fileMd5)

	// IE 6 just doesn't play nice. As dirty as this seems, it works.
	if c.Browser.IsIE6 && showImage {
		showImage = false
	} else if len(data) != 0 {
		// Guess the image content type from the bytes.
		ct := imageContentType(data)
		if ct != "" {
			h.Set("Content-Type", ct)
		} else if showImage {
			// Otherwise - let's think safety first... it might not be an
			// image...
			showImage = false
		}
	}

	disposition := "attachment"
	if showImage {
		disposition = "inline"
	}
	h.Set("Content-Disposition", disposition+`; filename="`+realFilename+`"`)
	if !showImage {
		h.Set("Content-Type", "application/octet-stream")
	}

	// If this has an "image extension" - but isn't actually an image - then
	// ensure it isn't cached cause of silly IE.
	tail := ""
	if len(realFilename) >= 4 {
		tail = realFilename[len(realFilename)-4:]
	}
	if !showImage && (tail == ".gif" || tail == ".jpg" || tail == ".bmp" || tail == ".png" || tail == "jpeg" || tail == "tiff") {
		h.Set("Cache-Control", "no-cache")
	} else {
		h.Set("Cache-Control", "max-age="+itoa(525600*60)+", private")
	}

	h.Set("Content-Length", itoa(len(data)))

	// For text files..... convert line endings per platform like PHP did.
	if !showImage && (tail == ".txt" || tail == ".css" || tail == ".htm" || tail == ".php" || tail == ".xml") {
		text := string(data)
		if strings.Contains(c.UserAgent, "Windows") {
			text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
		} else if strings.Contains(c.UserAgent, "Mac") {
			text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r")
		} else {
			text = strings.ReplaceAll(text, "\r", "\r\n")
		}
		data = []byte(text)
		h.Set("Content-Length", itoa(len(data)))
	}

	c.W.Write(data)
	c.exit()
}

// imageContentType sniffs the image type like getimagesize()'s mime field.
func imageContentType(data []byte) string {
	switch {
	case len(data) > 3 && data[0] == 'G' && data[1] == 'I' && data[2] == 'F':
		return "image/gif"
	case len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case len(data) > 8 && data[0] == 0x89 && string(data[1:4]) == "PNG":
		return "image/png"
	case len(data) > 2 && data[0] == 'B' && data[1] == 'M':
		return "image/x-ms-bmp"
	case len(data) > 4 && (string(data[:4]) == "II*\x00" || string(data[:4]) == "MM\x00*"):
		return "image/tiff"
	}
	return ""
}
