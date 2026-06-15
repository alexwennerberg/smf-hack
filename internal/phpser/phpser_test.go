package phpser

import "testing"

func TestRoundTrip(t *testing.T) {
	// The exact shape SMF's setLoginCookie produces:
	// serialize(array(ID_MEMBER, sha1(passwd.salt), time()+duration, cookie_state))
	in := []any{1, "356a192b7913b04c54574d18c28d46e6395428ab", 1735689600, 2}
	s, err := Serialize(in)
	if err != nil {
		t.Fatal(err)
	}
	want := `a:4:{i:0;i:1;i:1;s:40:"356a192b7913b04c54574d18c28d46e6395428ab";i:2;i:1735689600;i:3;i:2;}`
	if s != want {
		t.Errorf("Serialize = %q, want %q", s, want)
	}
	out, err := Unserialize(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 || out[0].(int64) != 1 || out[1].(string) != in[1].(string) || out[2].(int64) != 1735689600 || out[3].(int64) != 2 {
		t.Errorf("Unserialize = %#v", out)
	}
}

func TestLoggedOutCookie(t *testing.T) {
	// Logout sets serialize(array(0, '', 0)) — three elements.
	s, err := Serialize([]any{0, "", 0})
	if err != nil {
		t.Fatal(err)
	}
	if s != `a:3:{i:0;i:0;i:1;s:0:"";i:2;i:0;}` {
		t.Errorf("got %q", s)
	}
	out, err := Unserialize(s)
	if err != nil || len(out) != 3 {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestUnserializeRejectsGarbage(t *testing.T) {
	for _, bad := range []string{
		"", "a:1:{i:0;i:1;", `a:1:{s:1:"x";i:1;}`, `a:2:{i:0;i:1;}`,
		`a:1:{i:0;s:5:"ab";}`, `a:1:{i:0;O:8:"stdClass":0:{}}`,
		`a:1:{i:0;i:1;}x`,
	} {
		if _, err := Unserialize(bad); err == nil {
			t.Errorf("Unserialize(%q) succeeded, want error", bad)
		}
	}
}

func TestBinaryStringSafety(t *testing.T) {
	in := []any{0, "a\"b;c\x00d", 0}
	s, _ := Serialize(in)
	out, err := Unserialize(s)
	if err != nil {
		t.Fatal(err)
	}
	if out[1].(string) != "a\"b;c\x00d" {
		t.Errorf("got %q", out[1])
	}
}
