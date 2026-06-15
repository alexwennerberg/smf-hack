package app

// Phase 7: the registration center.

import (
	"net/url"
	"strings"
	"testing"
)

func TestRegCenterRenderAndReserve(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Register tab renders the form.
	_, body := get(t, a, "/index.php?action=regcenter;sa=register", admin)
	if !strings.Contains(body, `name="regSubmit"`) || !strings.Contains(body, `name="user"`) {
		t.Fatalf("admin register form missing:\n%.400s", body)
	}

	// Reserved names round-trip.
	sc, cookies := mbForm(t, a, "/index.php?action=regcenter;sa=reservednames", admin)
	w := postForm(t, a, "/index.php?action=regcenter", url.Values{
		"save_reserved_names": {"1"},
		"sa":                  {"reservednames"},
		"reserved":            {"admin\nroot"},
		"matchword":           {"on"},
		"sc":                  {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save reserved: status %d", w.Code)
	}
	if a.Setting("reserveNames") != "admin\nroot" || a.Setting("reserveWord") != "1" {
		t.Fatalf("reserved names not saved: %q / %q", a.Setting("reserveNames"), a.Setting("reserveWord"))
	}
}

func TestRegCenterSettings(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=regcenter;sa=settings", admin)
	w := postForm(t, a, "/index.php?action=regcenter", url.Values{
		"save":                     {"1"},
		"sa":                       {"settings"},
		"registration_method":      {"2"},
		"notify_new_registration":  {"on"},
		"password_strength":        {"1"},
		"visual_verification_type": {"3"},
		"coppaAge":                 {"0"},
		"coppaType":                {"0"},
		"coppaPost":                {""},
		"sc":                       {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("save reg settings: status %d body %.300s", w.Code, w.Body.String())
	}
	if a.Setting("registration_method") != "2" || a.Setting("password_strength") != "1" || a.Setting("disable_visual_verification") != "3" {
		t.Fatalf("reg settings not saved: method=%q strength=%q vv=%q",
			a.Setting("registration_method"), a.Setting("password_strength"), a.Setting("disable_visual_verification"))
	}
}

func TestAdminRegisterMember(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=regcenter;sa=register", admin)
	w, body := func() (int, string) {
		wr := postForm(t, a, "/index.php?action=regcenter", url.Values{
			"regSubmit": {"1"},
			"sa":        {"register"},
			"user":      {"zaphod"},
			"email":     {"zaphod@example.com"},
			"password":  {"secretpass"},
			"group":     {"0"},
			"sc":        {sc},
		}, cookies...)
		return wr.Code, wr.Body.String()
	}()
	if w != 200 {
		t.Fatalf("admin register: status %d", w)
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE memberName = 'zaphod'`)).Scan(&n)
	if n != 1 {
		t.Fatalf("member not created (count=%d)\n%.500s", n, body)
	}
	if !strings.Contains(body, "zaphod") {
		t.Errorf("registration done message missing")
	}
}
