package lang

import "testing"

// Counts cross-checked against grep -c '^\$txt\[' on the source files:
// index.english.php has 469 top-level entries, Errors.english.php 251.
// (The ICQ/AIM/YIM/MSN messenger entries were removed.)
func TestGeneratedCounts(t *testing.T) {
	idx := English["index"]
	n := len(idx.Strings) + len(idx.Dynamic) + len(idx.Lists)
	if n != 469 {
		t.Errorf("index entries = %d, want 469", n)
	}
	errs := English["Errors"]
	n = len(errs.Strings) + len(errs.Dynamic) + len(errs.Lists)
	if n != 251 {
		t.Errorf("Errors entries = %d, want 251", n)
	}
}

func TestKnownEntries(t *testing.T) {
	idx := English["index"]
	if idx.Strings["34"] != "Login" {
		t.Errorf("txt[34] = %q", idx.Strings["34"])
	}
	if idx.Dynamic["18"] != "{forum_name} - Index" {
		t.Errorf("txt[18] = %q", idx.Dynamic["18"])
	}
	if idx.Lists["months"].Base != 1 || idx.Lists["months"].Items[0] != "January" {
		t.Errorf("months = %#v", idx.Lists["months"])
	}
	if idx.Strings["lang_character_set"] != "ISO-8859-1" {
		t.Errorf("charset = %q", idx.Strings["lang_character_set"])
	}
	if idx.Strings["lang_rtl"] != "" {
		t.Errorf("lang_rtl = %q, want empty (false)", idx.Strings["lang_rtl"])
	}
	errs := English["Errors"]
	if errs.Dynamic["error_long_message"] != "The message exceeds the maximum allowed length ({modSettings:max_messageLength} characters)." {
		t.Errorf("error_long_message = %q", errs.Dynamic["error_long_message"])
	}
	if errs.Dynamic["profile_error_password_short"] != "Your password must be at least {password_min_chars} characters long." {
		t.Errorf("password_short override = %q", errs.Dynamic["profile_error_password_short"])
	}
}
