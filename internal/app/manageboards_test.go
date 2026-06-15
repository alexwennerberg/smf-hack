package app

// Phase 7: board/category administration.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// mbForm GETs a manageboards page and returns (sc, cookies) for posting back.
func mbForm(t *testing.T, a *App, path string, admin *http.Cookie) (string, []*http.Cookie) {
	t.Helper()
	w, body := get(t, a, path, admin)
	if w.Code != 200 {
		t.Fatalf("GET %s: status %d body %.300s", path, w.Code, body)
	}
	sc := scRe.FindStringSubmatch(body)
	if sc == nil {
		t.Fatalf("GET %s: no sc", path)
	}
	return sc[1], append([]*http.Cookie{admin}, cookiesFrom(w)...)
}

func TestManageBoardsList(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=manageboards", admin)
	// The seed board lives under category 1.
	if !strings.Contains(body, "General Discussion") {
		t.Errorf("board list missing seed board:\n%.500s", body)
	}
	if !strings.Contains(body, "?action=manageboards;sa=board;boardid=1") {
		t.Errorf("board list missing modify link")
	}
	if !strings.Contains(body, "?action=manageboards;sa=newcat") {
		// the tab.
	}
}

func TestManageBoardsCreateCategoryAndBoard(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Create a new category.
	sc, cookies := mbForm(t, a, "/index.php?action=manageboards;sa=newcat", admin)
	w := postForm(t, a, "/index.php?action=manageboards;sa=cat2", url.Values{
		"cat":      {"0"},
		"add":      {"Add"},
		"cat_name": {"My New Category"},
		"collapse": {"on"},
		"sc":       {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("create category: status %d body %.300s", w.Code, w.Body.String())
	}
	var catID int
	a.DB.QueryRow(a.Q(`SELECT ID_CAT FROM {$db_prefix}categories WHERE name = 'My New Category'`)).Scan(&catID)
	if catID == 0 {
		t.Fatalf("category not created")
	}

	// Create a board in that category.
	sc, cookies = mbForm(t, a, "/index.php?action=manageboards;sa=newboard;cat="+itoa(catID), admin)
	w = postForm(t, a, "/index.php?action=manageboards;sa=board2", url.Values{
		"boardid":         {"0"},
		"add":             {"Add"},
		"board_name":      {"My New Board"},
		"desc":            {"A shiny new board."},
		"cur_cat":         {itoa(catID)},
		"new_cat":         {"0"},
		"count":           {"on"},
		"boardtheme":      {"0"},
		"permission_mode": {"0"},
		"moderators":      {""},
		"sc":              {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("create board: status %d body %.300s", w.Code, w.Body.String())
	}
	var boardID, gotCat int
	var bname string
	a.DB.QueryRow(a.Q(`SELECT ID_BOARD, ID_CAT, name FROM {$db_prefix}boards WHERE name = 'My New Board'`)).Scan(&boardID, &gotCat, &bname)
	if boardID == 0 {
		t.Fatalf("board not created")
	}
	if gotCat != catID {
		t.Fatalf("board in category %d, want %d", gotCat, catID)
	}

	// The board appears in the list.
	_, body := get(t, a, "/index.php?action=manageboards", admin)
	if !strings.Contains(body, "My New Board") || !strings.Contains(body, "My New Category") {
		t.Errorf("new board/category missing from list")
	}

	// Edit the board's name.
	sc, cookies = mbForm(t, a, "/index.php?action=manageboards;sa=board;boardid="+itoa(boardID), admin)
	w = postForm(t, a, "/index.php?action=manageboards;sa=board2", url.Values{
		"boardid":         {itoa(boardID)},
		"edit":            {"Edit"},
		"board_name":      {"Renamed Board"},
		"desc":            {"A shiny new board."},
		"new_cat":         {"0"},
		"count":           {"on"},
		"boardtheme":      {"0"},
		"permission_mode": {"0"},
		"moderators":      {""},
		"no_children":     {"1"},
		"sc":              {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("edit board: status %d", w.Code)
	}
	a.DB.QueryRow(a.Q(`SELECT name FROM {$db_prefix}boards WHERE ID_BOARD = ?`), boardID).Scan(&bname)
	if bname != "Renamed Board" {
		t.Fatalf("board name = %q, want Renamed Board", bname)
	}

	// Delete the board (no children).
	sc, cookies = mbForm(t, a, "/index.php?action=manageboards;sa=board;boardid="+itoa(boardID), admin)
	w = postForm(t, a, "/index.php?action=manageboards;sa=board2", url.Values{
		"boardid":     {itoa(boardID)},
		"delete":      {"Delete"},
		"no_children": {"1"},
		"sc":          {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("delete board: status %d", w.Code)
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}boards WHERE ID_BOARD = ?`), boardID).Scan(&n)
	if n != 0 {
		t.Fatalf("board not deleted")
	}
}

func TestBoardSettingsRecycle(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=manageboards;sa=settings", admin)
	w := postForm(t, a, "/index.php?action=manageboards;sa=settings", url.Values{
		"save_settings":   {"1"},
		"countChildPosts": {"on"},
		"recycle_enable":  {"on"},
		"recycle_board":   {"1"},
		"sc":              {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save board settings: status %d", w.Code)
	}
	if a.Setting("recycle_board") != "1" || a.Setting("recycle_enable") != "1" || a.Setting("countChildPosts") != "1" {
		t.Fatalf("board settings not saved: recycle_board=%q recycle_enable=%q countChildPosts=%q",
			a.Setting("recycle_board"), a.Setting("recycle_enable"), a.Setting("countChildPosts"))
	}
}
