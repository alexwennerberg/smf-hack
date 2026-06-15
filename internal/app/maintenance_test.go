package app

// Phase 7: the maintenance center, log clearing, and board recount.

import (
	"strings"
	"testing"
)

func TestMaintainPage(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=maintain", admin)
	if !strings.Contains(body, "?action=boardrecount") {
		t.Errorf("maintain page missing recount link")
	}
	if !strings.Contains(body, "?action=removeoldtopics2") {
		t.Errorf("maintain page missing prune form")
	}
	// The prune board picker lists the seed board.
	if !strings.Contains(body, `name="boards[1]"`) {
		t.Errorf("maintain page missing board checkbox")
	}
}

func TestMaintainClearLogs(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_errors (logTime, ID_MEMBER, ip, url, message, session) VALUES (?, 1, '1.1.1.1', '?x', 'err', 's')`), nowUnix())
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_floodcontrol (ip, logTime) VALUES ('1.1.1.1', ?)`), nowUnix())

	w, _ := get(t, a, "/index.php?action=maintain;sa=logs", admin)
	if w.Code != 200 {
		t.Fatalf("maintain logs: status %d", w.Code)
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_errors`)).Scan(&n)
	if n != 0 {
		t.Errorf("error log not cleared (%d rows)", n)
	}
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_floodcontrol`)).Scan(&n)
	if n != 0 {
		t.Errorf("flood log not cleared (%d rows)", n)
	}
}

func TestBoardRecount(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	makeTopic(t, a, "Recount topic", admin)

	// Corrupt the board's counts.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numPosts = 999, numTopics = 999 WHERE ID_BOARD = 1`))

	// Actual message/topic counts for board 1.
	var realPosts, realTopics int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages WHERE ID_BOARD = 1`)).Scan(&realPosts)
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}topics WHERE ID_BOARD = 1`)).Scan(&realTopics)

	w, _ := get(t, a, "/index.php?action=boardrecount", admin)
	if w.Code != 302 {
		t.Fatalf("boardrecount: status %d body %.300s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "action=maintain;done") {
		t.Fatalf("boardrecount redirect %q, want maintain;done", loc)
	}

	var gotPosts, gotTopics int
	a.DB.QueryRow(a.Q(`SELECT numPosts, numTopics FROM {$db_prefix}boards WHERE ID_BOARD = 1`)).Scan(&gotPosts, &gotTopics)
	if gotPosts != realPosts || gotTopics != realTopics {
		t.Fatalf("recount got posts=%d topics=%d, want posts=%d topics=%d", gotPosts, gotTopics, realPosts, realTopics)
	}
}
