package app

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// App holds process-wide state: what PHP kept in globals like $modSettings,
// $scripturl and the database connection.
type App struct {
	DB     *sql.DB
	Config Config

	ScriptURL string // $scripturl = $boardurl . '/index.php'

	// $modSettings, loaded from the settings table (reloadSettings in
	// Load.php). Guarded by settingsMu; UpdateSettings keeps it in sync.
	settingsMu  sync.RWMutex
	modSettings map[string]string

	cache *memCache // cache_put_data/cache_get_data replacement
}

// Open opens (or creates) the SQLite database and loads settings. If the
// database file does not exist it is created with the default schema and
// data, mirroring what install.php did.
func Open(cfg Config) (*App, error) {
	_, statErr := os.Stat(cfg.DBPath)
	fresh := os.IsNotExist(statErr)

	// modernc.org/sqlite: enable WAL and busy timeout via pragmas. A single
	// writer at a time; MaxOpenConns>1 is fine for readers under WAL.
	dsn := "file:" + cfg.DBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(0)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	a := &App{
		DB:        db,
		Config:    cfg,
		ScriptURL: cfg.BoardURL + "/index.php",
		cache:     newMemCache(),
	}

	if fresh {
		if err := CreateSchema(db, cfg.DBPrefix); err != nil {
			db.Close()
			os.Remove(cfg.DBPath)
			return nil, err
		}
		if err := InsertDefaultData(db, cfg); err != nil {
			db.Close()
			os.Remove(cfg.DBPath)
			return nil, err
		}
	}

	if err := a.ReloadSettings(); err != nil {
		db.Close()
		return nil, err
	}
	return a, nil
}

// Tbl prefixes a table name: a.Tbl("members") -> "smf_members".
func (a *App) Tbl(name string) string {
	return a.Config.DBPrefix + name
}

// Q replaces {$db_prefix} in a query, so queries can be ported verbatim
// from the PHP sources.
func (a *App) Q(query string) string {
	return strings.ReplaceAll(query, "{$db_prefix}", a.Config.DBPrefix)
}

// ReloadSettings loads the settings table into memory ($modSettings).
// It also applies the few derived fixups reloadSettings() in Load.php does.
func (a *App) ReloadSettings() error {
	rows, err := a.DB.Query(a.Q(`SELECT variable, value FROM {$db_prefix}settings`))
	if err != nil {
		return err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		m[k] = v
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(m) == 0 {
		return fmt.Errorf("settings table is empty; database not installed?")
	}

	// Load.php: defaults applied after fetching.
	if empty(m["defaultMaxTopics"]) || atoi(m["defaultMaxTopics"]) <= 0 || atoi(m["defaultMaxTopics"]) > 999999 {
		m["defaultMaxTopics"] = "20"
	}
	if empty(m["defaultMaxMessages"]) || atoi(m["defaultMaxMessages"]) <= 0 || atoi(m["defaultMaxMessages"]) > 999999 {
		m["defaultMaxMessages"] = "15"
	}
	if empty(m["defaultMaxMembers"]) || atoi(m["defaultMaxMembers"]) <= 0 || atoi(m["defaultMaxMembers"]) > 999999 {
		m["defaultMaxMembers"] = "30"
	}

	a.settingsMu.Lock()
	a.modSettings = m
	a.settingsMu.Unlock()
	return nil
}

// Setting returns $modSettings[name], "" if unset (PHP would warn and treat
// as empty).
func (a *App) Setting(name string) string {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.modSettings[name]
}

// SettingInt returns the setting as an int, with PHP (int) cast semantics.
func (a *App) SettingInt(name string) int {
	return atoi(a.Setting(name))
}

// SettingEmpty mirrors empty($modSettings[name]): unset, "", "0" and 0 are
// all empty.
func (a *App) SettingEmpty(name string) bool {
	return empty(a.Setting(name))
}

// UpdateSettings is updateSettings() from Subs.php: writes changed values
// (REPLACE INTO) and refreshes the in-memory copy.
func (a *App) UpdateSettings(changes map[string]string) error {
	if len(changes) == 0 {
		return nil
	}
	tx, err := a.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range changes {
		if _, err := tx.Exec(a.Q(`REPLACE INTO {$db_prefix}settings (variable, value) VALUES (?, ?)`), k, v); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	a.settingsMu.Lock()
	for k, v := range changes {
		a.modSettings[k] = v
	}
	a.settingsMu.Unlock()
	return nil
}

// ---- PHP semantics helpers used everywhere ----

// atoi is PHP's (int) cast: parses leading digits (with optional sign),
// returns 0 on garbage. strconv.Atoi alone is too strict.
func atoi(s string) int {
	s = strings.TrimLeft(s, " \t\n\r\x00\x0B")
	end := 0
	for i, c := range s {
		if i == 0 && (c == '-' || c == '+') {
			end = i + 1
			continue
		}
		if c < '0' || c > '9' {
			break
		}
		end = i + 1
	}
	n, _ := strconv.Atoi(s[:end])
	return n
}

// empty mirrors PHP empty() for strings: "", "0" are empty.
func empty(s string) bool {
	return s == "" || s == "0"
}
