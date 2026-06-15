package app

// Port of the request dispatch in index.php (smf_main). Actions are filled
// in phase by phase; unported actions fall through to the default like an
// unknown action does in SMF (BoardIndex).

import (
	"compress/gzip"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handler is a ported action function, e.g. (a) BoardIndex.
type handler func(c *Ctx)

// actionArray mirrors $actionArray in index.php. Populated in init() blocks
// as each source file is ported; see registerAction calls in each file.
var actionArray = map[string]handler{}

func registerAction(name string, h handler) {
	actionArray[name] = h
}

func init() {
	// Placeholders until the Phase 1 controllers land; routed through the
	// normal template pipeline so the chrome can be golden-diffed early.
	placeholder := func(name string) handler {
		return func(c *Ctx) {
			c.PageTitle = name
			c.SubTemplate = func(c *Ctx) {
				c.O("\n<div>", name, " not yet ported.</div>")
			}
		}
	}
	if _, ok := actionArray["*boardindex"]; !ok {
		registerAction("*boardindex", placeholder("BoardIndex"))
	}
	if _, ok := actionArray["*messageindex"]; !ok {
		registerAction("*messageindex", placeholder("MessageIndex"))
	}
	if _, ok := actionArray["*display"]; !ok {
		registerAction("*display", placeholder("Display"))
	}
}

// ServeHTTP routes a request: /index.php (or /) goes through the SMF
// dispatcher, everything else is a static asset under assetdir.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/index.go", a.dispatch)
	mux.HandleFunc("/{$}", a.dispatch)
	mux.HandleFunc("/", a.serveStatic)
	return mux
}

func (a *App) dispatch(w http.ResponseWriter, r *http.Request) {
	c := NewCtx(a, w, r)
	if c == nil {
		return // request was rejected (bad variables)
	}

	// Mirror PHP's shutdown behavior: the session is written and the output
	// buffer flushed however the request ends (normal return, fatal error,
	// redirect — all panic with smfExit after preparing c.Out).
	defer func() {
		if rec := recover(); rec != nil {
			if _, ok := rec.(smfExit); !ok {
				panic(rec)
			}
		}
		c.saveSession()
		if c.Out.Len() > 0 {
			out := c.Out.Bytes()
			// ob_gzhandler(): compress the buffered page when the admin enabled
			// it and the client advertised gzip. Binary responses (dlattach,
			// captcha) write to w directly and never reach this buffer.
			if c.gzipResponse() {
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Add("Vary", "Accept-Encoding")
				w.Header().Del("Content-Length")
				gz := gzip.NewWriter(w)
				gz.Write(out)
				gz.Close()
			} else {
				w.Write(out)
			}
		}
	}()

	c.loadSession()
	c.smfMain()

	// Call obExit specially; we're coming from the main area ;).
	c.obExit(nil, nil, true)
}

// gzipResponse reports whether the buffered HTML/XML output should be gzip'd,
// mirroring index.php's ob_start('ob_gzhandler') gate.
func (c *Ctx) gzipResponse() bool {
	if c.App.SettingEmpty("enableCompressedOutput") {
		return false
	}
	if !strings.Contains(c.R.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}
	if c.W.Header().Get("Content-Encoding") != "" {
		return false
	}
	ct := c.W.Header().Get("Content-Type")
	return strings.HasPrefix(ct, "text/html") || strings.HasPrefix(ct, "text/xml")
}

// smfMain is smf_main() from index.php.
func (c *Ctx) smfMain() {
	a := c.App

	// Special case: session keep-alive.
	if c.GET.Str("action") == "keepalive" {
		c.exit()
	}

	// Load the user's cookie (or set as guest) and load their settings.
	c.loadUserSettings()

	// Load the current board's information.
	c.loadBoard()

	// Load the current theme. (note that ?theme=1 will also work.)
	c.loadTheme()

	// Check if the user should be disallowed access.
	c.isNotBanned(false)

	// Load the current user's permissions.
	c.loadPermissions()

	action := c.REQUEST.Str("action")

	// Do some logging, unless this is an attachment, avatar, theme option or
	// XML feed.
	if action == "" || (action != "dlattach" && action != "jsoption" && action != ".xml") {
		// Log this user as online.
		c.writeLog(false)

		// Track forum statistics and hits...?
		if !a.SettingEmpty("hitStats") {
			c.trackStatsHit()
		}
	}

	// Is the forum in maintenance mode? (doesn't apply to administrators.)
	if a.Config.Maintenance != 0 && !c.allowedTo("admin_forum") {
		// You can only login.... otherwise, you're getting the "maintenance
		// mode" display.
		if action == "login2" || action == "logout" {
			if h, ok := actionArray[action]; ok {
				h(c)
				return
			}
		}
		c.inMaintenanceScreen()
		return
	}

	// If guest access is off, a guest can only do one of the very few
	// following actions.
	if a.SettingEmpty("allow_guestAccess") && c.User.IsGuest {
		allowed := map[string]bool{"coppa": true, "login": true, "login2": true, "register": true,
			"register2": true, "reminder": true, "activate": true, "smstats": true, "help": true,
			"verificationcode": true}
		if !allowed[action] {
			c.kickGuest()
			return
		}
	}

	if action == "" {
		// Action and board are both empty... BoardIndex!
		switch {
		case c.Board == 0 && c.Topic == 0:
			actionArray["*boardindex"](c)
		case c.Topic == 0:
			actionArray["*messageindex"](c)
		default:
			actionArray["*display"](c)
		}
		return
	}

	h, ok := actionArray[action]
	if !ok {
		// Fall through to the board index then...
		actionArray["*boardindex"](c)
		return
	}
	h(c)
}

// inMaintenanceScreen is InMaintenance() from Subs-Auth.php: the login form
// with the maintenance message (full port arrives with Phase 2's Login
// template; the chrome and message behavior match).
func (c *Ctx) inMaintenanceScreen() {
	c.loadLanguage("Login")
	c.PageTitle = c.App.Config.MTitle
	c.SubTemplate = func(c *Ctx) {
		c.O(`
<div align="center">
	<table border="0" cellspacing="1" cellpadding="4" class="tborder" align="center" width="80%">
		<tr class="titlebg">
			<td>`, c.App.Config.MTitle, `</td>
		</tr><tr class="windowbg">
			<td>`, c.App.Config.MMessage, `</td>
		</tr>
	</table>
</div>`)
	}
	c.obExit(nil, nil, false)
}

// kickGuest is KickGuest() from Subs-Auth.php.
func (c *Ctx) kickGuest() {
	c.loadLanguage("Login")
	c.PageTitle = c.Txt("34")
	c.KickMessage = ""
	c.SubTemplate = templateKickGuest
	c.obExit(nil, nil, false)
}

// serveStatic serves theme assets, smileys and avatars from assetdir.
// Attachments are NOT served here — they go through ?action=dlattach with
// permission checks, as in PHP (the attachments dir was .htaccess-protected).
func (a *App) serveStatic(w http.ResponseWriter, r *http.Request) {
	upath := path_Clean(r.URL.Path)

	// Only expose the directories a stock SMF install exposed over HTTP.
	allowed := false
	for _, pre := range []string{"/Themes/", "/Smileys/", "/avatars/"} {
		if strings.HasPrefix(upath, pre) {
			allowed = true
			break
		}
	}
	if !allowed || strings.Contains(upath, "/.") {
		http.NotFound(w, r)
		return
	}

	full := filepath.Join(a.Config.AssetDir, filepath.FromSlash(upath))
	st, err := os.Stat(full)
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, full)
}

// path_Clean normalizes the URL path, refusing traversal.
func path_Clean(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// Reject any dot-dot segments outright.
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "/"
		}
	}
	return p
}
