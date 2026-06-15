package app

// Port of loadSession() and the sessionOpen/Read/Write/Destroy/GC handlers
// from Load.php. Sessions live in the {$db_prefix}sessions table exactly as
// in PHP, except the data column holds JSON instead of PHP's session
// encoding (documented schema deviation; no data migration is needed).
//
// The cookie keeps PHP's default name PHPSESSID — Who's Online and the
// cookie-less fallbacks reference it, and parity is cheap.

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

const sessionCookieName = "PHPSESSID"

// Session is $_SESSION. Values are kept in a JSON-compatible map; numeric
// values decode as float64 and are converted by the typed accessors.
type Session struct {
	ID    string
	data  map[string]any
	dirty bool
	isNew bool
}

func (s *Session) Get(key string) any { return s.data[key] }

func (s *Session) GetStr(key string) string {
	v, _ := s.data[key].(string)
	return v
}

func (s *Session) GetInt(key string) int64 {
	switch v := s.data[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

func (s *Session) Has(key string) bool {
	_, ok := s.data[key]
	return ok
}

func (s *Session) Set(key string, v any) {
	s.data[key] = v
	s.dirty = true
}

func (s *Session) Delete(key string) {
	if _, ok := s.data[key]; ok {
		delete(s.data, key)
		s.dirty = true
	}
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func validSessionID(id string) bool {
	if len(id) < 16 || len(id) > 32 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}

// loadSession starts (or resumes) the session for this request, setting the
// cookie on new sessions and ensuring rand_code ($sc, the 'sesc' token).
func (c *Ctx) loadSession() {
	a := c.App
	var id string
	if cookie, err := c.R.Cookie(sessionCookieName); err == nil && validSessionID(cookie.Value) {
		id = cookie.Value
	}

	s := &Session{data: map[string]any{}}
	if id != "" {
		var data string
		err := a.DB.QueryRow(a.Q(`SELECT data FROM {$db_prefix}sessions WHERE session_id = ? LIMIT 1`), id).Scan(&data)
		if err == nil {
			s.ID = id
			if json.Unmarshal([]byte(data), &s.data) != nil {
				s.data = map[string]any{}
			}
		} else if err == sql.ErrNoRows {
			// Stale cookie: keep the ID like PHP does (session_start adopts
			// the supplied ID) so the cookie doesn't churn.
			s.ID = id
			s.isNew = true
		}
	}
	if s.ID == "" {
		s.ID = newSessionID()
		s.isNew = true
	}
	if s.isNew {
		http.SetCookie(c.W, &http.Cookie{
			Name:     sessionCookieName,
			Value:    s.ID,
			Path:     "/",
			HttpOnly: true,
		})
		s.dirty = true
	}

	// Loose caching, as in loadSession() when databaseSession_loose is on.
	if !a.SettingEmpty("databaseSession_loose") {
		c.W.Header().Set("Cache-Control", "private")
	}

	// Set the randomly generated code.
	if !s.Has("rand_code") {
		s.Set("rand_code", newSessionID())
	}
	c.Session = s
	c.Sc = s.GetStr("rand_code")
}

// saveSession persists the session (sessionWrite + occasional sessionGC).
func (c *Ctx) saveSession() {
	if c.Session == nil || !c.Session.dirty {
		return
	}
	a := c.App
	data, err := json.Marshal(c.Session.data)
	if err != nil {
		return
	}
	now := time.Now().Unix()
	res, _ := a.DB.Exec(a.Q(`UPDATE {$db_prefix}sessions SET data = ?, last_update = ? WHERE session_id = ?`),
		string(data), now, c.Session.ID)
	if res != nil {
		if n, _ := res.RowsAffected(); n == 0 {
			a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}sessions (session_id, data, last_update) VALUES (?, ?, ?)`),
				c.Session.ID, string(data), now)
		}
	}

	// GC roughly 1% of the time, like PHP's session.gc_probability.
	if now%100 == 0 {
		maxLifetime := int64(a.SettingInt("databaseSession_lifetime"))
		if maxLifetime < 60 {
			maxLifetime = 1440
		}
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}sessions WHERE last_update < ?`), now-maxLifetime)
	}
}
