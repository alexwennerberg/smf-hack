package app

// Port of the membergroup helpers in Sources/Subs-Members.php: deleteMembergroups,
// removeMembersFromGroups, addMembersToGroup. MySQL CONCAT/IF rewritten to SQLite
// (||/IIF); FIND_IN_SET is a registered SQLite function.

import "strings"

// csvDiff removes the given values from a comma-separated list.
func csvDiff(csv string, remove []int) string {
	rm := map[int]bool{}
	for _, r := range remove {
		rm[r] = true
	}
	var out []string
	for _, p := range strings.Split(csv, ",") {
		if p == "" {
			continue
		}
		if !rm[atoi(p)] {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

// findInSetAny builds "(FIND_IN_SET(g1, col) OR FIND_IN_SET(g2, col) ...)".
func findInSetAny(col string, groups []int) string {
	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = "FIND_IN_SET(" + itoa(g) + ", " + col + ")"
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

// deleteMembergroups is deleteMembergroups($groups).
func (c *Ctx) deleteMembergroups(groups []int) bool {
	a := c.App
	c.isAllowedTo("manage_membergroups")

	// Protected groups: guests, members, admins, moderators, newbies.
	protected := map[int]bool{-1: true, 0: true, 1: true, 3: true, 4: true}
	var del []int
	for _, g := range uniqueInts(groups) {
		if !protected[g] {
			del = append(del, g)
		}
	}
	if len(del) == 0 {
		return false
	}
	in := joinInts(del)

	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}membergroups WHERE ID_GROUP IN (` + in + `)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE ID_GROUP IN (` + in + `)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}board_permissions WHERE ID_GROUP IN (` + in + `)`))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_GROUP = 0 WHERE ID_GROUP IN (` + in + `)`))

	// Additional groups.
	c.rewriteCSVColumn("members", "ID_MEMBER", "additionalGroups", del)
	// Board access lists.
	c.rewriteCSVColumn("boards", "ID_BOARD", "memberGroups", del)

	c.updateStatsPostgroups(0)
	return true
}

// rewriteCSVColumn removes the given group IDs from a CSV column on every row
// of the table that references any of them.
func (c *Ctx) rewriteCSVColumn(table, idCol, csvCol string, groups []int) {
	a := c.App
	rows, err := a.DB.Query(a.Q(`SELECT ` + idCol + `, ` + csvCol + ` FROM {$db_prefix}` + table + ` WHERE ` + findInSetAny(csvCol, groups)))
	if err != nil {
		return
	}
	type upd struct {
		id  int
		csv string
	}
	var updates []upd
	for rows.Next() {
		var u upd
		rows.Scan(&u.id, &u.csv)
		updates = append(updates, u)
	}
	rows.Close()
	for _, u := range updates {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}`+table+` SET `+csvCol+` = ? WHERE `+idCol+` = ?`), csvDiff(u.csv, groups), u.id)
	}
}

// removeMembersFromGroups is removeMembersFromGroups($members, $groups). Pass
// groups = nil to remove the members from all groups.
func (c *Ctx) removeMembersFromGroups(members []int, groups []int, allGroups bool) bool {
	a := c.App
	c.isAllowedTo("manage_membergroups")

	members = uniqueInts(members)
	if len(members) == 0 {
		return false
	}
	memberIn := joinInts(members)

	if allGroups {
		extra := ""
		if !c.allowedTo("admin_forum") {
			extra = " AND ID_GROUP != 1 AND NOT FIND_IN_SET(1, additionalGroups)"
		}
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_GROUP = 0, additionalGroups = '' WHERE ID_MEMBER IN (` + memberIn + `)` + extra))
		c.updateStatsPostgroups(0)
		return true
	}

	// Implicit groups can't be removed explicitly.
	implicit := map[int]bool{-1: true, 0: true, 3: true}
	prows, err := a.DB.Query(a.Q(`SELECT ID_GROUP FROM {$db_prefix}membergroups WHERE minPosts != -1`))
	if err == nil {
		for prows.Next() {
			var g int
			prows.Scan(&g)
			implicit[g] = true
		}
		prows.Close()
	}
	var del []int
	for _, g := range uniqueInts(groups) {
		if implicit[g] {
			continue
		}
		if g == 1 && !c.allowedTo("admin_forum") {
			continue
		}
		del = append(del, g)
	}
	if len(del) == 0 {
		return false
	}
	in := joinInts(del)

	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_GROUP = 0 WHERE ID_GROUP IN (` + in + `) AND ID_MEMBER IN (` + memberIn + `)`))

	rows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER, additionalGroups FROM {$db_prefix}members WHERE ` + findInSetAny("additionalGroups", del) + ` AND ID_MEMBER IN (` + memberIn + `)`))
	if err == nil {
		type upd struct {
			id  int
			csv string
		}
		var updates []upd
		for rows.Next() {
			var u upd
			rows.Scan(&u.id, &u.csv)
			updates = append(updates, u)
		}
		rows.Close()
		for _, u := range updates {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET additionalGroups = ? WHERE ID_MEMBER = ?`), csvDiff(u.csv, del), u.id)
		}
	}

	c.updateStatsPostgroups(0)
	return true
}

// addMembersToGroup is addMembersToGroup($members, $group, $type).
func (c *Ctx) addMembersToGroup(members []int, group int, typ string) bool {
	a := c.App
	c.isAllowedTo("manage_membergroups")

	members = uniqueInts(members)

	implicit := map[int]bool{-1: true, 0: true, 3: true}
	prows, err := a.DB.Query(a.Q(`SELECT ID_GROUP FROM {$db_prefix}membergroups WHERE minPosts != -1`))
	if err == nil {
		for prows.Next() {
			var g int
			prows.Scan(&g)
			implicit[g] = true
		}
		prows.Close()
	}
	if implicit[group] || len(members) == 0 {
		return false
	}
	if group == 1 && !c.allowedTo("admin_forum") {
		return false
	}
	in := joinInts(members)
	g := itoa(group)

	switch typ {
	case "only_additional":
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}members
			SET additionalGroups = IIF(additionalGroups = '', '` + g + `', additionalGroups || ',` + g + `')
			WHERE ID_MEMBER IN (` + in + `) AND ID_GROUP != ` + g + ` AND NOT FIND_IN_SET(` + g + `, additionalGroups)`))
	case "only_primary", "force_primary":
		extra := ""
		if typ != "force_primary" {
			extra = " AND ID_GROUP = 0 AND NOT FIND_IN_SET(" + g + ", additionalGroups)"
		}
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_GROUP = ` + g + ` WHERE ID_MEMBER IN (` + in + `)` + extra))
	case "auto":
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}members
			SET additionalGroups = IIF(ID_GROUP = 0, additionalGroups, IIF(additionalGroups = '', '` + g + `', additionalGroups || ',` + g + `')),
				ID_GROUP = IIF(ID_GROUP = 0, ` + g + `, ID_GROUP)
			WHERE ID_MEMBER IN (` + in + `) AND ID_GROUP != ` + g + ` AND NOT FIND_IN_SET(` + g + `, additionalGroups)`))
	default:
		return false
	}
	return true
}
