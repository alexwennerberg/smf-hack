package strftime

import (
	"testing"
	"time"
)

// Expected values generated from glibc via GNU date (TZ=UTC, en_US):
//   date -d @1718000000 '+...'

func TestFormatFixtures(t *testing.T) {
	utc := time.Unix(1718000000, 0).UTC() // Mon Jun 10 06:13:20 UTC 2024
	tests := []struct{ format, want string }{
		{"%a|%A|%b|%B", "Mon|Monday|Jun|June"},
		{"%C|%d|%D|%e", "20|10|06/10/24|10"},
		{"%g|%G|%H|%I|%j", "24|2024|06|06|162"},
		{"%k|%l", " 6| 6"},
		{"%m|%M|%p|%P", "06|13|AM|am"},
		{"%r|%R|%S|%T", "06:13:20 AM|06:13|20|06:13:20"},
		{"%u|%U|%V|%w|%W", "1|23|24|1|24"},
		{"%y|%Y|%z|%Z", "24|2024|+0000|UTC"},
		{"%c", "Mon 10 Jun 2024 06:13:20 AM UTC"},
		{"%x|%X", "06/10/2024|06:13:20 AM"},
		{"%%|%n|%t", "%|\n|\t"},
		{"%q", "%q"}, // unknown specifier passes through
		// SMF's default time_format:
		{"%B %d, %Y, %I:%M:%S %p", "June 10, 2024, 06:13:20 AM"},
	}
	for _, tt := range tests {
		if got := Format(tt.format, utc, nil); got != tt.want {
			t.Errorf("Format(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestWeekNumbersEdges(t *testing.T) {
	// 2000-01-01 (Saturday) and 2024-12-31 (Tuesday), fixtures from date(1).
	jan1 := time.Unix(946684800, 0).UTC()
	if got := Format("%U|%W|%V|%G|%j|%w", jan1, nil); got != "00|00|52|1999|001|6" {
		t.Errorf("jan1 = %q", got)
	}
	dec31 := time.Unix(1735689599, 0).UTC()
	if got := Format("%U|%W|%V|%G", dec31, nil); got != "52|53|01|2025" {
		t.Errorf("dec31 = %q", got)
	}
}

func TestPMAndNoon(t *testing.T) {
	noon := time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)
	if got := Format("%I %p", noon, nil); got != "12 PM" {
		t.Errorf("noon = %q", got)
	}
	midnight := time.Date(2024, 6, 10, 0, 5, 0, 0, time.UTC)
	if got := Format("%I:%M %p", midnight, nil); got != "12:05 AM" {
		t.Errorf("midnight = %q", got)
	}
}

func TestCustomNames(t *testing.T) {
	// SMF substitutes $txt day/month names when the locale is missing.
	n := English
	n.Months[5] = "juin"
	d := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	if got := Format("%B", d, &n); got != "juin" {
		t.Errorf("got %q", got)
	}
}
