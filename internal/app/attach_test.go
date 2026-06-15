package app

// Attachment upload / download / delete flow tests (Phase 3).

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// postMultipart posts a multipart form with one file field "attachment[]".
func postMultipart(t *testing.T, a *App, path string, fields map[string]string, filename string, fileData []byte, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		w.WriteField(k, v)
	}
	if filename != "" {
		fw, _ := w.CreateFormFile("attachment[]", filename)
		fw.Write(fileData)
	}
	w.Close()

	r := httptest.NewRequest("POST", "http://127.0.0.1:8080"+path, &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	if i := strings.IndexByte(path, '?'); i >= 0 {
		u, _ := url.Parse("http://127.0.0.1:8080" + path[:i])
		u.RawQuery = path[i+1:]
		r.URL = u
	}
	for _, ck := range cookies {
		r.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, r)
	return rec
}

func TestAttachmentUploadDownload(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	os.MkdirAll(a.Setting("attachmentUploadDir"), 0755)

	// Build a small PNG (bigger than the 150px thumb threshold? keep small,
	// no thumbnail expected).
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{uint8(40 * x), uint8(40 * y), 128, 255})
		}
	}
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)

	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)

	w := postMultipart(t, a, "/index.php?action=post2;start=0;board=1.0", map[string]string{
		"topic": "0", "subject": "Attachment topic", "message": "Has a file.",
		"icon": "xx", "additional_options": "0", "sc": sc, "seqnum": seq,
	}, "test.png", pngBuf.Bytes(), cookies...)
	if w.Code != 302 {
		t.Fatalf("post2 with attachment: status %d body %.400s", w.Code, w.Body.String())
	}

	// The attachment row exists and is linked to the new message.
	var attachID, msgID, size int
	var filename string
	err := a.DB.QueryRow(a.Q(`SELECT ID_ATTACH, ID_MSG, size, filename FROM {$db_prefix}attachments WHERE attachmentType = 0 ORDER BY ID_ATTACH DESC LIMIT 1`)).
		Scan(&attachID, &msgID, &size, &filename)
	if err != nil || msgID == 0 {
		t.Fatalf("no attachment row: %v", err)
	}
	if filename != "test.png" || size != pngBuf.Len() {
		t.Errorf("attachment row: name=%q size=%d want test.png/%d", filename, size, pngBuf.Len())
	}

	var topicID int
	a.DB.QueryRow(a.Q(`SELECT ID_TOPIC FROM {$db_prefix}messages WHERE ID_MSG = ?`), msgID).Scan(&topicID)
	topic := itoa(topicID)

	// It shows up on the topic display.
	_, body := get(t, a, "/index.php?topic="+topic+".0", admin)
	if !strings.Contains(body, "test.png") {
		t.Error("attachment not shown on topic display")
	}

	// Download it.
	w2, dlBody := get(t, a, "/index.php?action=dlattach;topic="+topic+".0;attach="+itoa(attachID), admin)
	if w2.Code != 200 {
		t.Fatalf("dlattach: status %d", w2.Code)
	}
	if dlBody != pngBuf.String() {
		t.Errorf("downloaded bytes differ: got %d bytes want %d", len(dlBody), pngBuf.Len())
	}
	var downloads int
	a.DB.QueryRow(a.Q(`SELECT downloads FROM {$db_prefix}attachments WHERE ID_ATTACH = ?`), attachID).Scan(&downloads)
	if downloads != 1 {
		t.Errorf("downloads = %d, want 1", downloads)
	}

	// Edit the message unchecking the attachment: it gets removed.
	clearFloodControl(a)
	w3, body := get(t, a, "/index.php?action=post;topic="+topic+".0", admin)
	sc = scRe.FindStringSubmatch(body)[1]
	cookies = append([]*http.Cookie{admin}, cookiesFrom(w3)...)
	editPath := "/index.php?action=post;msg=" + itoa(msgID) + ";topic=" + topic + ".0;sesc=" + sc
	sc, seq, cookies = openPostForm(t, a, editPath, cookies...)

	w = postForm(t, a, "/index.php?action=post2;start=0;msg="+itoa(msgID)+";board=1.0", url.Values{
		"topic": {topic}, "subject": {"Attachment topic"}, "message": {"Has a file."},
		"icon": {"xx"}, "additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
		"attach_del[]": {"0"},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("edit removing attachment: status %d body %.300s", w.Code, w.Body.String())
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}attachments WHERE ID_ATTACH = ?`), attachID).Scan(&n)
	if n != 0 {
		t.Errorf("attachment row still present after delete")
	}
}

func TestThumbnailCreation(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	os.MkdirAll(a.Setting("attachmentUploadDir"), 0755)

	// A 300x200 PNG exceeds the 150x150 thumbnail limit.
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 300; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 99, 255})
		}
	}
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)

	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postMultipart(t, a, "/index.php?action=post2;start=0;board=1.0", map[string]string{
		"topic": "0", "subject": "Thumb topic", "message": "Big image.",
		"icon": "xx", "additional_options": "0", "sc": sc, "seqnum": seq,
	}, "big.png", pngBuf.Bytes(), cookies...)
	if w.Code != 302 {
		t.Fatalf("post2: status %d body %.300s", w.Code, w.Body.String())
	}

	// Parent + thumb rows; PHP floor() math: 300x200 -> 150x100.
	var thumbW, thumbH, idThumb int
	err := a.DB.QueryRow(a.Q(`SELECT width, height, ID_ATTACH FROM {$db_prefix}attachments WHERE attachmentType = 3`)).
		Scan(&thumbW, &thumbH, &idThumb)
	if err != nil {
		t.Fatalf("no thumbnail row: %v", err)
	}
	if thumbW != 150 || thumbH != 100 {
		t.Errorf("thumb dimensions = %dx%d, want 150x100", thumbW, thumbH)
	}
	var linked int
	a.DB.QueryRow(a.Q(`SELECT ID_THUMB FROM {$db_prefix}attachments WHERE attachmentType = 0`)).Scan(&linked)
	if linked != idThumb {
		t.Errorf("parent ID_THUMB = %d, want %d", linked, idThumb)
	}
}

// TestAttachmentRejectCleansTempFile guards the upload_tmp_* leak: a rejected
// attachment (bad extension) must not leave its staged temp file behind.
func TestAttachmentRejectCleansTempFile(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	dir := a.Setting("attachmentUploadDir")
	os.MkdirAll(dir, 0755)
	// Turn on extension checking and disallow .exe.
	a.UpdateSettings(map[string]string{
		"attachmentEnable":          "1",
		"attachmentCheckExtensions": "1",
		"attachmentExtensions":      "png,jpg,gif,txt",
	})

	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postMultipart(t, a, "/index.php?action=post2;start=0;board=1.0", map[string]string{
		"topic": "0", "subject": "Bad ext", "message": "x",
		"icon": "xx", "additional_options": "0", "sc": sc, "seqnum": seq,
	}, "evil.exe", []byte("MZ malware"), cookies...)
	// Rejected -> the error page (200), not a 302 redirect.
	if w.Code == 302 {
		t.Fatalf("disallowed .exe was accepted (status 302)")
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "upload_tmp_") {
			t.Errorf("leaked staged temp file after rejected upload: %s", e.Name())
		}
	}
}
