package app

// The createThumbnail/resizeImage subset of Sources/Subs-Graphics.php,
// reimplemented on Go's image packages (gif/jpeg/png; PHP additionally
// handled bmp/wbmp via GD). The dimension math matches PHP exactly; the
// re-encoded pixel bytes do not need to.

import (
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

// urlImageSize is a port of url_image_size($url) (Subs.php): fetch a remote
// image with a short socket timeout and return its (width, height). Returns
// ok=false on any failure — unreachable host, non-200 response, timeout, or a
// body that isn't a decodable image — exactly as PHP's url_image_size() returns
// false, leaving callers to fall back to their no-dimensions path. Only http(s)
// URLs are honoured (PHP's fetch_web_data also did ftp; no call site needs it).
// The body read is capped so a hostile server can't stream forever; image
// headers carry the dimensions in the first few KB regardless.
func urlImageSize(url string) (int, int, bool) {
	low := strings.ToLower(url)
	if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
		return 0, 0, false
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, false
	}
	// PHP's fetch_web_data() sends this exact User-Agent.
	req.Header.Set("User-Agent", "PHP/SMF")
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()
	// PHP only proceeds past a 200/201 status line.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, 0, false
	}
	cfg, _, err := image.DecodeConfig(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// imageSize returns (width, height, ok) like getimagesize().
func imageSize(path string) (int, int, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// createThumbnail is createThumbnail($source, $max_width, $max_height):
// write a resized copy to source+"_thumb".
func (a *App) createThumbnail(source string, maxWidth, maxHeight int) bool {
	if maxWidth == 0 && maxHeight == 0 {
		return false
	}

	f, err := os.Open(source)
	if err != nil {
		return false
	}
	srcImg, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		// PHP touches an empty thumb file on failure.
		os.WriteFile(source+"_thumb", nil, 0644)
		return false
	}

	destName := source + "_thumb"
	if a.resizeImage(srcImg, destName+".tmp", maxWidth, maxHeight) &&
		os.Rename(destName+".tmp", destName) == nil {
		return true
	}
	os.Remove(destName + ".tmp")
	os.WriteFile(destName, nil, 0644)
	return false
}

// resizeImage is resizeImage(): scale to fit (max_width, max_height)
// preserving aspect ratio, then encode (JPEG quality 65 unless
// avatar_download_png).
func (a *App) resizeImage(srcImg image.Image, destName string, maxWidth, maxHeight int) bool {
	srcWidth := srcImg.Bounds().Dx()
	srcHeight := srcImg.Bounds().Dy()

	dstImg := srcImg
	if maxWidth != 0 || maxHeight != 0 {
		var dstW, dstH int
		if maxWidth != 0 && (maxHeight == 0 || srcHeight*maxWidth/srcWidth <= maxHeight) {
			dstW = maxWidth
			dstH = int(math.Floor(float64(srcHeight) * float64(maxWidth) / float64(srcWidth)))
		} else if maxHeight != 0 {
			dstW = int(math.Floor(float64(srcWidth) * float64(maxHeight) / float64(srcHeight)))
			dstH = maxHeight
		}

		// Don't bother resizing if it's already smaller...
		if dstW != 0 && dstH != 0 && (dstW < srcWidth || dstH < srcHeight) {
			dstImg = scaleBilinear(srcImg, dstW, dstH)
		}
	}

	out, err := os.Create(destName)
	if err != nil {
		return false
	}
	defer out.Close()

	if !a.SettingEmpty("avatar_download_png") {
		return png.Encode(out, dstImg) == nil
	}
	return jpeg.Encode(out, dstImg, &jpeg.Options{Quality: 65}) == nil
}

// scaleBilinear is a simple bilinear scaler (stand-in for GD's
// imagecopyresampled).
func scaleBilinear(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	for y := 0; y < h; y++ {
		fy := (float64(y) + 0.5) * float64(sh) / float64(h)
		y0 := int(fy - 0.5)
		if y0 < 0 {
			y0 = 0
		}
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		wy := fy - 0.5 - float64(y0)
		if wy < 0 {
			wy = 0
		}
		for x := 0; x < w; x++ {
			fx := (float64(x) + 0.5) * float64(sw) / float64(w)
			x0 := int(fx - 0.5)
			if x0 < 0 {
				x0 = 0
			}
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			wx := fx - 0.5 - float64(x0)
			if wx < 0 {
				wx = 0
			}

			mix := func(c0, c1 uint32, t float64) float64 {
				return float64(c0)*(1-t) + float64(c1)*t
			}
			r00, g00, b00, a00 := src.At(sb.Min.X+x0, sb.Min.Y+y0).RGBA()
			r10, g10, b10, a10 := src.At(sb.Min.X+x1, sb.Min.Y+y0).RGBA()
			r01, g01, b01, a01 := src.At(sb.Min.X+x0, sb.Min.Y+y1).RGBA()
			r11, g11, b11, a11 := src.At(sb.Min.X+x1, sb.Min.Y+y1).RGBA()

			r := mix(r00, r10, wx)*(1-wy) + mix(r01, r11, wx)*wy
			g := mix(g00, g10, wx)*(1-wy) + mix(g01, g11, wx)*wy
			b := mix(b00, b10, wx)*(1-wy) + mix(b01, b11, wx)*wy
			al := mix(a00, a10, wx)*(1-wy) + mix(a01, a11, wx)*wy

			dst.Set(x, y, color.RGBA64{uint16(r), uint16(g), uint16(b), uint16(al)})
		}
	}
	return dst
}

// Ensure the decoders are linked even if nothing else imports them.
var _ = gif.Decode
