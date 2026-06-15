// Package strftime implements C strftime(3) formatting with glibc en_US
// semantics, as PHP's strftime() exposes it on Linux — which is what SMF's
// timeformat() (Sources/Subs.php) feeds user-configurable format strings to.
//
// Day/month names are parameterized: SMF substitutes %a/%A/%b/%B itself from
// $txt before calling strftime when the system locale is unavailable, and
// uses the locale otherwise; for English both agree with these defaults.
package strftime

import (
	"fmt"
	"strings"
	"time"
)

type Names struct {
	Days        [7]string  // Sunday..Saturday
	DaysShort   [7]string  // Sun..Sat
	Months      [12]string // January..December
	MonthsShort [12]string
	AM, PM      string
}

var English = Names{
	Days:        [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
	DaysShort:   [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
	Months:      [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	MonthsShort: [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	AM:          "AM",
	PM:          "PM",
}

// Format renders t according to the strftime format string.
func Format(format string, t time.Time, n *Names) string {
	if n == nil {
		n = &English
	}
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			b.WriteByte(format[i])
			continue
		}
		i++
		writeSpec(&b, format[i], t, n)
	}
	return b.String()
}

func writeSpec(b *strings.Builder, spec byte, t time.Time, n *Names) {
	switch spec {
	case 'a':
		b.WriteString(n.DaysShort[int(t.Weekday())])
	case 'A':
		b.WriteString(n.Days[int(t.Weekday())])
	case 'b', 'h':
		b.WriteString(n.MonthsShort[int(t.Month())-1])
	case 'B':
		b.WriteString(n.Months[int(t.Month())-1])
	case 'c': // glibc en_US: "%a %d %b %Y %r %Z"
		writeSpec(b, 'a', t, n)
		b.WriteByte(' ')
		writeSpec(b, 'd', t, n)
		b.WriteByte(' ')
		writeSpec(b, 'b', t, n)
		b.WriteByte(' ')
		writeSpec(b, 'Y', t, n)
		b.WriteByte(' ')
		writeSpec(b, 'r', t, n)
		b.WriteByte(' ')
		writeSpec(b, 'Z', t, n)
	case 'C':
		fmt.Fprintf(b, "%02d", t.Year()/100)
	case 'd':
		fmt.Fprintf(b, "%02d", t.Day())
	case 'D':
		fmt.Fprintf(b, "%02d/%02d/%02d", int(t.Month()), t.Day(), t.Year()%100)
	case 'e':
		fmt.Fprintf(b, "%2d", t.Day())
	case 'F':
		fmt.Fprintf(b, "%04d-%02d-%02d", t.Year(), int(t.Month()), t.Day())
	case 'g':
		y, _ := t.ISOWeek()
		fmt.Fprintf(b, "%02d", y%100)
	case 'G':
		y, _ := t.ISOWeek()
		fmt.Fprintf(b, "%d", y)
	case 'H':
		fmt.Fprintf(b, "%02d", t.Hour())
	case 'I':
		fmt.Fprintf(b, "%02d", hour12(t))
	case 'j':
		fmt.Fprintf(b, "%03d", t.YearDay())
	case 'k':
		fmt.Fprintf(b, "%2d", t.Hour())
	case 'l':
		fmt.Fprintf(b, "%2d", hour12(t))
	case 'm':
		fmt.Fprintf(b, "%02d", int(t.Month()))
	case 'M':
		fmt.Fprintf(b, "%02d", t.Minute())
	case 'n':
		b.WriteByte('\n')
	case 'p':
		if t.Hour() < 12 {
			b.WriteString(n.AM)
		} else {
			b.WriteString(n.PM)
		}
	case 'P':
		if t.Hour() < 12 {
			b.WriteString(strings.ToLower(n.AM))
		} else {
			b.WriteString(strings.ToLower(n.PM))
		}
	case 'r': // "%I:%M:%S %p"
		fmt.Fprintf(b, "%02d:%02d:%02d ", hour12(t), t.Minute(), t.Second())
		writeSpec(b, 'p', t, n)
	case 'R':
		fmt.Fprintf(b, "%02d:%02d", t.Hour(), t.Minute())
	case 's':
		fmt.Fprintf(b, "%d", t.Unix())
	case 'S':
		fmt.Fprintf(b, "%02d", t.Second())
	case 't':
		b.WriteByte('\t')
	case 'T':
		fmt.Fprintf(b, "%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
	case 'u': // ISO weekday, Monday=1..Sunday=7
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		fmt.Fprintf(b, "%d", wd)
	case 'U': // week of year, first Sunday starts week 1
		fmt.Fprintf(b, "%02d", (t.YearDay()+6-int(t.Weekday()))/7)
	case 'V':
		_, w := t.ISOWeek()
		fmt.Fprintf(b, "%02d", w)
	case 'w':
		fmt.Fprintf(b, "%d", int(t.Weekday()))
	case 'W': // week of year, first Monday starts week 1
		wd := (int(t.Weekday()) + 6) % 7 // Monday=0
		fmt.Fprintf(b, "%02d", (t.YearDay()+6-wd)/7)
	case 'x': // glibc en_US: "%m/%d/%Y"
		fmt.Fprintf(b, "%02d/%02d/%04d", int(t.Month()), t.Day(), t.Year())
	case 'X': // glibc en_US: "%r"
		writeSpec(b, 'r', t, n)
	case 'y':
		fmt.Fprintf(b, "%02d", t.Year()%100)
	case 'Y':
		fmt.Fprintf(b, "%d", t.Year())
	case 'z':
		_, off := t.Zone()
		sign := '+'
		if off < 0 {
			sign = '-'
			off = -off
		}
		fmt.Fprintf(b, "%c%02d%02d", sign, off/3600, off%3600/60)
	case 'Z':
		zone, _ := t.Zone()
		b.WriteString(zone)
	case '%':
		b.WriteByte('%')
	default:
		// Unknown specifier: glibc emits it verbatim including the '%'.
		b.WriteByte('%')
		b.WriteByte(spec)
	}
}

func hour12(t time.Time) int {
	h := t.Hour() % 12
	if h == 0 {
		h = 12
	}
	return h
}
