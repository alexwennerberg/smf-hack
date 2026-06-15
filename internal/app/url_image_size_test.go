package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

// servePNG starts a test server returning a single PNG of the given size.
func servePNG(t *testing.T, w, h int) (url string, hits *int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{1, 2, 3, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	body := buf.Bytes()
	n := 0
	hits = &n
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		n++
		rw.Header().Set("Content-Type", "image/png")
		rw.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/pic.png", hits
}

func TestUrlImageSize(t *testing.T) {
	url, _ := servePNG(t, 200, 120)

	w, h, ok := urlImageSize(url)
	if !ok || w != 200 || h != 120 {
		t.Fatalf("urlImageSize(%q) = %d,%d,%v; want 200,120,true", url, w, h, ok)
	}

	// Non-http schemes and unreachable hosts return ok=false (PHP returns false).
	if _, _, ok := urlImageSize("ftp://example.com/x.png"); ok {
		t.Error("ftp URL should not be sized")
	}
	if _, _, ok := urlImageSize("http://127.0.0.1:1/nope.png"); ok {
		t.Error("unreachable host should return ok=false")
	}
}

func TestPreparseImageResize(t *testing.T) {
	c := newTestCtx(t)
	url, hits := servePNG(t, 200, 100)

	if err := c.App.UpdateSettings(map[string]string{
		"max_image_width":  "100",
		"max_image_height": "0",
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, in, want string
	}{
		{
			// No dimensions: fetch real 200x100, clamp width to 100, scale height.
			name: "no dims",
			in:   "[img]" + url + "[/img]",
			want: "[img width=100 height=50]" + url + "[/img]",
		},
		{
			// Explicit oversized width, height omitted -> derive then clamp.
			name: "explicit width",
			in:   "[img width=300]" + url + "[/img]",
			want: "[img width=100 height=50]" + url + "[/img]",
		},
		{
			// Already within the cap -> left untouched.
			name: "within cap",
			in:   "[img width=80 height=40]" + url + "[/img]",
			want: "[img width=80 height=40]" + url + "[/img]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.preparsecode(tc.in, true)
			if got != tc.want {
				t.Errorf("preparsecode(%q):\n got %q\nwant %q", tc.in, got, tc.want)
			}
		})
	}
	if *hits == 0 {
		t.Error("expected the image to be fetched at least once")
	}
}

func TestPreparseImageResizeDisabled(t *testing.T) {
	c := newTestCtx(t)
	url, hits := servePNG(t, 999, 999)

	// With both caps at 0 (defaults), the block is skipped entirely — no fetch,
	// no rewrite.
	in := "[img]" + url + "[/img]"
	if got := c.preparsecode(in, true); got != in {
		t.Errorf("with no caps the tag should be untouched; got %q", got)
	}
	if *hits != 0 {
		t.Errorf("no remote fetch should happen when caps are unset; got %d hits", *hits)
	}
}

func TestPreparseImageResizeFetchFails(t *testing.T) {
	c := newTestCtx(t)
	if err := c.App.UpdateSettings(map[string]string{
		"max_image_width":  "100",
		"max_image_height": "100",
	}); err != nil {
		t.Fatal(err)
	}
	// Unreachable URL, no explicit dims -> urlImageSize fails -> dims stay 0 ->
	// the tag is left unchanged (PHP's list(,) = false path).
	in := "[img]http://127.0.0.1:1/x.png[/img]"
	if got := c.preparsecode(in, true); got != in {
		t.Errorf("failed fetch should leave the tag intact; got %q", got)
	}
}
