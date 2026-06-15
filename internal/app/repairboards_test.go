package app

// Phase 7: forum integrity checker (pure-SQL subset).

import (
	"strings"
	"testing"
)

func TestRepairBoardsDetectAndFix(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// A topic with deliberately wrong stats, plus one message in it.
	res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}topics (ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, numReplies) VALUES (1, 0, 0, 7)`))
	tid64, _ := res.LastInsertId()
	tid := int(tid64)
	mres, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterTime, subject, body) VALUES (?, 1, 1, ?, 'x', 'y')`), tid, nowUnix())
	mid64, _ := mres.LastInsertId()
	mid := int(mid64)

	// A topic on a board that does not exist (with a message, so it isn't
	// pruned as an empty topic before the board salvage runs).
	ores, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}topics (ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, numReplies) VALUES (999, 0, 0, 0)`))
	otid64, _ := ores.LastInsertId()
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterTime, subject, body) VALUES (?, 999, 1, ?, 'o', 'p')`), int(otid64), nowUnix())

	// Detection lists problems and offers the fix link.
	sc, cookies := mbForm(t, a, "/index.php?action=maintain", admin)
	_, body := get(t, a, "/index.php?action=repairboards", cookies...)
	if !strings.Contains(body, "action=repairboards;fixErrors") {
		t.Fatalf("detection did not offer a fix link:\n%.500s", body)
	}

	// Apply the fixes.
	_, body = get(t, a, "/index.php?action=repairboards;fixErrors;sesc="+sc, cookies...)
	_ = body

	// Topic stats corrected: one message -> 0 replies, first/last = that message.
	var replies, first, last int
	a.DB.QueryRow(a.Q(`SELECT numReplies, ID_FIRST_MSG, ID_LAST_MSG FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), tid).Scan(&replies, &first, &last)
	if replies != 0 || first != mid || last != mid {
		t.Fatalf("topic stats not fixed: replies=%d first=%d last=%d (msg=%d)", replies, first, last, mid)
	}

	// The orphaned-board topic was salvaged onto a new, real board.
	var orphanCount int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}topics WHERE ID_BOARD = 999`)).Scan(&orphanCount)
	if orphanCount != 0 {
		t.Fatalf("orphaned-board topic was not salvaged")
	}
	var salvageCats int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}categories WHERE name = 'Salvage Area'`)).Scan(&salvageCats)
	if salvageCats == 0 {
		t.Errorf("salvage category was not created")
	}
}

func TestRepairBoardsNoErrors(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=repairboards", admin)
	if !strings.Contains(body, "action=maintain") {
		t.Fatalf("clean forum should show the return link:\n%.400s", body)
	}
}
