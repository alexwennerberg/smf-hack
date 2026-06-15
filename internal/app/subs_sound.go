package app

// Port of createWaveFile() from Sources/Subs-Sound.php: spells out the
// captcha code by concatenating per-letter wave samples with random
// distortion. The distortion uses Go's math/rand instead of PHP's seeded
// mt_rand — the output is random noise either way.

import (
	"encoding/binary"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// createWaveFile is createWaveFile($word).
func (c *Ctx) createWaveFile(word string) bool {
	a := c.App

	// Allow max 2 requests per 20 seconds.
	count := 0
	if v, ok := a.cache.Get("wave_file/" + c.User.IP); ok {
		count = v.(int)
	}
	if count > 2 {
		c.W.WriteHeader(400)
		c.exit()
	}
	a.cache.Put("wave_file/"+c.User.IP, count+1, 20*time.Second)

	soundDir := filepath.Join(a.Config.AssetDir, "Themes/default/fonts/sound")

	// Try to see if there's a sound font in the user's language.
	soundLanguage := c.User.Language
	if _, err := os.Stat(filepath.Join(soundDir, "a."+soundLanguage+".wav")); err != nil {
		// English should be there.
		if _, err := os.Stat(filepath.Join(soundDir, "a.english.wav")); err == nil {
			soundLanguage = "english"
		} else {
			return false
		}
	}

	word = strings.ToLower(word)

	var soundWord []byte
	for i := 0; i < len(word); i++ {
		letter := word[i : i+1]
		data, err := os.ReadFile(filepath.Join(soundDir, letter+"."+soundLanguage+".wav"))
		if err != nil {
			return false
		}
		idx := strings.Index(string(data), "data")
		if idx < 0 {
			return false
		}
		sample := data[idx+8:]

		mode := rand.Intn(3)
		if letter == "s" {
			mode = 0
		}
		switch mode {
		case 0:
			for _, b := range sample {
				m := (rand.Intn(11) + 15 + 5) / 10 // round(rand(15,25)/10)
				for k := 0; k < m; k++ {
					if letter == "s" {
						soundWord = append(soundWord, b)
					} else {
						lo, hi := int(b)-1, int(b)+1
						if lo < 0 {
							lo = 0
						}
						if hi > 255 {
							hi = 255
						}
						soundWord = append(soundWord, byte(lo+rand.Intn(hi-lo+1)))
					}
				}
			}
		case 1:
			n := len(sample) - 1
			for j := 0; j < n; j += 2 {
				if rand.Intn(4) != 0 {
					soundWord = append(soundWord, sample[j])
				}
				if rand.Intn(4) == 0 {
					soundWord = append(soundWord, sample[j+1])
				} else {
					soundWord = append(soundWord, sample[j])
				}
				if rand.Intn(4) == 0 {
					soundWord = append(soundWord, sample[j])
				} else {
					soundWord = append(soundWord, sample[j+1])
				}
				soundWord = append(soundWord, sample[j+1])
				if rand.Intn(4) == 0 {
					soundWord = append(soundWord, sample[j+1])
				}
			}
			soundWord = append(soundWord, sample[n], sample[n])
		case 2:
			shift := 0
			for _, b := range sample {
				if rand.Intn(11) == 0 {
					shift += rand.Intn(7) - 3
				}
				m := (rand.Intn(11) + 15 + 5) / 10
				for k := 0; k < m; k++ {
					v := int(b) + shift
					if v < 0 {
						v = 0
					}
					if v > 255 {
						v = 255
					}
					soundWord = append(soundWord, byte(v))
				}
			}
		}

		pad := 10000 + rand.Intn(501)
		for k := 0; k < pad; k++ {
			soundWord = append(soundWord, 0x80)
		}
	}

	dataSize := len(soundWord)
	fileSize := dataSize + 0x24
	sampleRate := uint32(16000)

	// Output the wav (the RIFF/WAVE header from PHP's pack format).
	h := c.W.Header()
	h.Set("Content-Type", "audio/x-wav")
	h.Set("Expires", time.Now().Add(525600*60*time.Second).UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	h.Set("Content-Length", itoa(fileSize+0x08))

	var hdr []byte
	be16 := func(v uint16) { hdr = binary.BigEndian.AppendUint16(hdr, v) }
	le32 := func(v uint32) { hdr = binary.LittleEndian.AppendUint32(hdr, v) }
	be16(0x5249) // 'RI'
	be16(0x4646) // 'FF'
	le32(uint32(fileSize))
	be16(0x5741) // 'WA'
	be16(0x5645) // 'VE'
	be16(0x666D) // 'fm'
	be16(0x7420) // 't '
	be16(0x1000)
	be16(0x0000)
	be16(0x0100)
	be16(0x0100)
	le32(sampleRate)
	le32(sampleRate)
	be16(0x0100)
	be16(0x0800)
	be16(0x6461) // 'da'
	be16(0x7461) // 'ta'
	le32(uint32(dataSize))

	c.Out.Write(hdr)
	c.Out.Write(soundWord)
	c.exit()
	return true
}
