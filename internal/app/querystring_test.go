package app

import "testing"

func TestParseQueryString(t *testing.T) {
	tests := []struct {
		raw  string
		want map[string]string
	}{
		// Standard SMF ';' URLs.
		{"action=admin;area=manageboards", map[string]string{"action": "admin", "area": "manageboards"}},
		{"topic=123.15", map[string]string{"topic": "123.15"}},
		{"action=profile;u=1;sa=showPosts", map[string]string{"action": "profile", "u": "1", "sa": "showPosts"}},
		// Search engines mangle ';' into '%3B'; the first urldecode pass fixes it.
		{"action=profile%3Bu=1", map[string]string{"action": "profile", "u": "1"}},
		// Valueless flags get '='.
		{"action=post;board=1.0;poll", map[string]string{"action": "post", "board": "1.0", "poll": ""}},
		{"debug;board=2", map[string]string{"debug": "", "board": "2"}},
		// Chains of valueless flags.
		{"a=1;x;y;z", map[string]string{"a": "1", "x": "", "y": "", "z": ""}},
		// '&' works too.
		{"board=1&action=who", map[string]string{"board": "1", "action": "who"}},
		// ';?' is treated like '&'.
		{"wwwRedirect;?board=1", map[string]string{"wwwRedirect": "", "board": "1"}},
		// %00 and NUL stripped.
		{"action=pro%00file", map[string]string{"action": "profile"}},
		// PHP key mangling: dots in key names become underscores.
		{"a.b=1", map[string]string{"a_b": "1"}},
	}
	for _, tt := range tests {
		got := ParseQueryString(tt.raw)
		for k, want := range tt.want {
			if !got.Has(k) {
				t.Errorf("%q: missing key %q", tt.raw, k)
			} else if got.Str(k) != want {
				t.Errorf("%q: key %q = %q, want %q", tt.raw, k, got.Str(k), want)
			}
		}
		if got.Len() != len(tt.want) {
			t.Errorf("%q: got %d keys %v, want %d", tt.raw, got.Len(), got.Keys(), len(tt.want))
		}
	}
}

func TestParseQueryStringArrays(t *testing.T) {
	got := ParseQueryString("topics[]=5;topics[]=9;options[abc]=1")
	topics := got.Arr("topics")
	if topics == nil || topics.Len() != 2 || topics.Str("0") != "5" || topics.Str("1") != "9" {
		t.Fatalf("topics = %#v", topics)
	}
	options := got.Arr("options")
	if options == nil || options.Str("abc") != "1" {
		t.Fatalf("options = %#v", options)
	}
}

func TestHtmlspecialchars(t *testing.T) {
	if got := Htmlspecialchars(`<a href="x">&'`); got != "&lt;a href=&quot;x&quot;&gt;&amp;&#039;" {
		t.Errorf("got %q", got)
	}
}

func TestPHPIntCast(t *testing.T) {
	for in, want := range map[string]int{
		"123": 123, "123abc": 123, "abc": 0, "": 0, "-5": -5,
		"12.9": 12, " 42": 42, "+7": 7,
	} {
		if got := atoi(in); got != want {
			t.Errorf("atoi(%q) = %d, want %d", in, got, want)
		}
	}
}
