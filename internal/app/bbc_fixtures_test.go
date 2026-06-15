package app

// PHP-authoritative BBC parity: testdata/bbc_fixtures.txt holds
// base64(input)\tbase64(output) records captured from the REAL SMF 1.1.21
// parse_bbc() running on php:5.6 (see tools/parity/bbc_gen.go + bbc_harness.php).
// This replaces hand-derived expectations with the genuine oracle. Regenerate
// after a deliberate parse_bbc change by re-running the generator against the
// live PHP reference.

import (
	"bufio"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseBBCFixtures(t *testing.T) {
	// The PHP reference ran in UTC with a guest (time_offset 0); match it so the
	// one dated input ([time]/quote date=) formats identically.
	time.Local = time.UTC

	f, err := os.Open(filepath.Join("testdata", "bbc_fixtures.txt"))
	if err != nil {
		t.Fatalf("missing bbc_fixtures.txt (generate with tools/parity/bbc_gen.go): %v", err)
	}
	defer f.Close()

	c := newTestCtx(t)

	var total, fails int
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			t.Fatalf("bad fixture line: %.60s", line)
		}
		inB, err1 := base64.StdEncoding.DecodeString(parts[0])
		wantB, err2 := base64.StdEncoding.DecodeString(parts[1])
		if err1 != nil || err2 != nil {
			t.Fatalf("bad base64 in fixture: %v %v", err1, err2)
		}
		in, want := string(inB), string(wantB)
		total++
		if got := c.parseBBC(in, true); got != want {
			fails++
			t.Errorf("parseBBC mismatch (input %q)\n  php: %q\n  go:  %q", in, want, got)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("BBC fixtures: %d total, %d mismatch", total, fails)
}
