package app

// Ctx is the per-request state: what PHP kept in $_GET/$_POST/$_REQUEST,
// $board, $topic, $context, and the response output buffer. One Ctx is
// created per HTTP request and threaded through controllers and templates.

import (
	"bytes"
	"net/http"
	"regexp"
	"strings"
	"time"

	"smf/internal/lang"
)

type Ctx struct {
	App *App
	W   http.ResponseWriter
	R   *http.Request

	// Request data after cleanRequest(): GET values are htmlspecialchars'd,
	// REQUEST is POST merged over GET (POST wins, like $_POST + $_GET).
	GET     *PArray
	POST    *PArray
	REQUEST *PArray

	Board int // $board
	Topic int // $topic
	Start int // $_REQUEST['start']

	IP         string // $_SERVER['REMOTE_ADDR'] after proxy resolution
	BanCheckIP string // $_SERVER['BAN_CHECK_IP']
	UserAgent  string // $_SERVER['HTTP_USER_AGENT'], htmlspecialchars'd
	RequestURL string // $_SERVER['REQUEST_URL']

	// Output buffer: pages render here and are flushed once at the end
	// (obExit), so error pages can discard partial output.
	Out bytes.Buffer

	timeStart time.Time // $time_start, for the footer load time

	// Loaded by loadSession/loadUserSettings/loadBoard/loadTheme:
	Session   *Session
	Sc        string // $sc / $context['session_id'] — the 'sesc' token
	User      *User  // $user_info
	BoardInfo *BoardInfo
	Theme     *ThemeSettings    // $settings (theme)
	Options   map[string]string // $options (user's theme options)

	langFiles []*lang.File // $txt include stack

	// ---- $context chrome fields consumed by tpl_index.go ----
	PageTitle          string
	HTMLHeaders        string // $context['html_headers']
	LinkTree           []Link
	TemplateLayers     []string
	SubTemplate        func(*Ctx) // $context['sub_template'] (nil = 'main')
	CharacterSet       string
	MenuSeparator      string
	CurrentAction      string
	CurrentSubaction   string
	RobotNoIndex       bool
	Browser            Browser
	NewsLines          []string
	FaderNewsLines     []string
	RandomNewsLine     string
	InMaintenance      bool
	CurrentTime        string
	ShowQuickLogin     bool
	AllowSearch        bool
	AllowAdmin         bool
	AllowEditProfile   bool
	AllowMemberlist    bool
	AllowPM            bool
	UnapprovedMembers  int
	DisableLoginHash   bool // $context['disable_login_hashing']
	PopupMessages      bool
	UserAvatar         UserAvatarCtx // $context['user']['avatar']
	TimeLoggedIn       struct{ Days, Hours, Minutes int }
	buttonCache        map[string]string // PHP's $buttons global (template_button_strip)
	CommonStats        CommonStats
	NoLastModified     bool // $context['no_last_modified']
	JumpTo             []JumpToCat
	KickMessage        string   // $context['kick_message'] (kick_guest)
	ShowLoadTime       bool     // $context['show_load_time']
	LoadTime           string   // $context['load_time']
	LoadQueries        int      // $context['load_queries']
	WelcomeGuest       string   // $txt['welcome_guest'] (+activate hint)
	RightToLeft        bool     // $context['right_to_left']
	FormSequenceNumber int      // $context['form_sequence_number']
	PostErrors         []string // $context['post_error'] keys

	// Admin chrome ($context['admin_areas'] / admin_section / admin_area /
	// admin_tabs), set by adminIndex() and consumed by the 'admin' layer.
	AdminAreas   []AdminSection
	AdminSection string
	AdminArea    string
	AdminTabs    *AdminTabs

	// Current ManageSmileys sub-action (drives which sub-template/context to build).
	smileySubAction string

	// ManageAttachments: which tab is highlighted by the manage_files layer.
	attachSelected string

	// ShowHelp: $context['current_page'] — the manual page URL key (index,
	// registering, ...), consumed by the 'manual' layer's menu.
	helpCurrentPage string

	// Reports table-builder state (newTable/addData/setKeys/finishTables).
	reports    *reportBuilder
	reportType string

	// Per-request modSettings overrides (e.g. derived values like the split
	// pm_spam_settings) — read by settings forms, never persisted.
	settingOverride map[string]string

	// Inline group-permission UI (ManagePermissions init/save/theme): permission
	// -> ordered group rows, plus the current permission for the template.
	InlinePermissions    map[string][]InlinePermGroup
	CurrentPermission    string
	CanChangePermissions bool
	illegalPermissions   []string

	// Page-specific context struct, set by the controller and consumed by
	// its template (e.g. *BoardIndexCtx).
	Page any

	// $user_profile / $memberContext (members.go).
	memberProfiles map[int]*memberProfile
	memberCtx      map[int]*MemberCtx

	headerDone bool // obExit statics
	footerDone bool
}

// Link is one linktree entry; ExtraBefore/ExtraAfter are raw HTML shown
// around the link (used by Display for the page index, etc.).
type Link struct {
	URL, Name               string
	ExtraBefore, ExtraAfter string
}

type UserAvatarCtx struct {
	Href, Image   string
	Width, Height string // emitted only if non-empty
}

type CommonStats struct {
	TotalPosts, TotalTopics, TotalMembers string
	LatestMemberID                        int
	LatestMemberName                      string
	LatestMemberHref, LatestMemberLink    string
}

type JumpToCat struct {
	ID     int
	Name   string
	Boards []JumpToBoard
}

type JumpToBoard struct {
	ID         int
	Name       string
	ChildLevel int
	IsCurrent  bool
}

// Browser is $context['browser'] from loadTheme().
type Browser struct {
	IsOpera, IsOpera6, IsOpera7, IsOpera8       bool
	IsIE4, IsIE5, IsIE55, IsIE6, IsIE7, IsIE    bool
	IsMacIE, IsWebTV, IsKonqueror, IsGecko      bool
	IsSafari, IsFirefox, IsFirefox1, IsFirefox2 bool
	NeedsSizeFix, PossiblyRobot                 bool
}

// O writes its arguments to the output buffer like PHP echo: strings as-is,
// ints in decimal. Templates use this for every echo statement.
func (c *Ctx) O(args ...any) {
	for _, a := range args {
		switch v := a.(type) {
		case string:
			c.Out.WriteString(v)
		case int:
			c.Out.WriteString(itoa(v))
		case int64:
			c.Out.WriteString(itoa(int(v)))
		default:
			panic("Ctx.O: unsupported type")
		}
	}
}

// selAttr returns ` selected="selected"` when b is true, for <option> tags.
func selAttr(b bool) string {
	if b {
		return ` selected="selected"`
	}
	return ""
}

// joinStr joins xs with sep (like strings.Join).
func joinStr(xs []string, sep string) string {
	out := ""
	for i, s := range xs {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

var ipRe = regexp.MustCompile(`^((([1]?\d)?\d|2[0-4]\d|25[0-5])\.){3}(([1]?\d)?\d|2[0-4]\d|25[0-5])$`)
var privateIPRe = regexp.MustCompile(`^((0|10|172\.(1[6-9]|2[0-9]|3[01])|192\.168|255|127)\.|unknown)`)

// NewCtx parses the request the way cleanRequest() does and returns the
// per-request context.
func NewCtx(a *App, w http.ResponseWriter, r *http.Request) *Ctx {
	c := &Ctx{App: a, W: w, R: r, timeStart: time.Now()}

	// $_GET: parsed with the ';' pipeline, then htmlspecialchars on values.
	c.GET = ParseQueryString(r.URL.RawQuery)
	HtmlspecialcharsRecursive(c.GET)

	// $_POST: application/x-www-form-urlencoded or multipart. Raw values —
	// PHP only addslashes'd POST (a no-op semantically with bound
	// parameters); htmlspecialchars happens at the use sites.
	c.POST = NewPArray()
	if r.Method == http.MethodPost {
		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/form-data") {
			r.ParseMultipartForm(32 << 20)
			if r.MultipartForm != nil {
				for key, vals := range r.MultipartForm.Value {
					for _, v := range vals {
						base, path := splitBrackets(key)
						setNested(c.POST, base, path, v)
					}
				}
			}
		} else {
			r.ParseForm()
			for key, vals := range r.PostForm {
				for _, v := range vals {
					base, path := splitBrackets(key)
					setNested(c.POST, base, path, v)
				}
			}
		}
	}

	// These keys shouldn't be set... ever.
	if c.GET.Has("GLOBALS") || c.POST.Has("GLOBALS") {
		http.Error(w, "Invalid request variable.", http.StatusBadRequest)
		return nil
	}
	// Same goes for numeric keys in GET/POST.
	for _, k := range append(append([]string{}, c.GET.Keys()...), c.POST.Keys()...) {
		if isAllDigits(k) {
			http.Error(w, "Invalid request variable.", http.StatusBadRequest)
			return nil
		}
	}

	// $_REQUEST = $_POST + $_GET (left side wins in PHP's + operator).
	c.REQUEST = NewPArray()
	c.POST.Values(func(k string, v any) { c.REQUEST.Set(k, v) })
	c.GET.Values(func(k string, v any) {
		if !c.REQUEST.Has(k) {
			c.REQUEST.Set(k, v)
		}
	})

	// Make sure $board and $topic are numbers.
	if c.REQUEST.Has("board") {
		board := c.REQUEST.Str("board")
		// '?board=1/15' (old links) or '?board=1.15' both carry a start value.
		if i := strings.IndexAny(board, "/."); i >= 0 {
			c.REQUEST.Set("start", board[i+1:])
			board = board[:i]
		}
		c.Board = atoi(board)
		c.REQUEST.Set("board", itoa(c.Board))
		c.GET.Set("board", itoa(c.Board))
	}

	// Old YaBB SE links use threadid.
	if c.REQUEST.Has("threadid") && !c.REQUEST.Has("topic") {
		c.REQUEST.Set("topic", c.REQUEST.Str("threadid"))
	}
	if c.REQUEST.Has("topic") {
		topic := c.REQUEST.Str("topic")
		if i := strings.IndexAny(topic, "/."); i >= 0 {
			c.REQUEST.Set("start", topic[i+1:])
			topic = topic[:i]
		}
		c.Topic = atoi(topic)
		c.GET.Set("topic", itoa(c.Topic))
	}

	start := c.REQUEST.Str("start")
	c.Start = atoi(start)
	if empty(start) || c.Start < 0 || c.Start > 2147473647 {
		c.Start = 0
	}
	// Keep the raw start when it carries a Display deep-link prefix
	// (new/msg<id>/from<time>); SMF leaves $_REQUEST['start'] untouched and
	// each consumer (int)-casts it. Only normalize the plain numeric form.
	if isAllDigits(start) || start == "" {
		c.REQUEST.Set("start", itoa(c.Start))
	}

	c.resolveIP()

	// $_SERVER['REQUEST_URL']
	if m := regexp.MustCompile(`^([^/]+//[^/]+)`).FindStringSubmatch(a.ScriptURL); m != nil {
		c.RequestURL = m[1] + r.URL.RequestURI()
	} else {
		c.RequestURL = r.URL.RequestURI()
	}

	c.UserAgent = Htmlspecialchars(r.Header.Get("User-Agent"))

	return c
}

// resolveIP ports the REMOTE_ADDR / X-Forwarded-For / Client-IP logic.
func (c *Ctx) resolveIP() {
	remoteAddr := c.R.RemoteAddr
	if i := strings.LastIndexByte(remoteAddr, ':'); i >= 0 {
		remoteAddr = remoteAddr[:i]
	}
	remoteAddr = strings.Trim(remoteAddr, "[]") // IPv6 brackets

	// BAN_CHECK_IP: REMOTE_ADDR if it is a well-formed IPv4, else 'unknown'.
	if ipRe.MatchString(remoteAddr) {
		c.BanCheckIP = remoteAddr
	} else {
		c.BanCheckIP = "unknown"
	}

	forwardedFor := c.R.Header.Get("X-Forwarded-For")
	clientIP := c.R.Header.Get("Client-Ip")

	if forwardedFor != "" && clientIP != "" &&
		(!privateIPRe.MatchString(clientIP) || privateIPRe.MatchString(remoteAddr)) {
		// We have both forwarded for AND client IP... check the first
		// forwarded for as the block - only switch if it's better that way.
		ffBlock := firstToken(forwardedFor, '.')
		ciBlock := firstToken(clientIP, '.')
		if ffBlock != ciBlock && "."+ffBlock == lastChunk(clientIP, '.') &&
			(!privateIPRe.MatchString(forwardedFor) || privateIPRe.MatchString(remoteAddr)) {
			remoteAddr = reverseDots(clientIP)
		} else {
			remoteAddr = clientIP
		}
	}
	if clientIP != "" && (!privateIPRe.MatchString(clientIP) || privateIPRe.MatchString(remoteAddr)) {
		// Since they are in different blocks, it's probably reversed.
		if firstToken(remoteAddr, '.') != firstToken(clientIP, '.') {
			remoteAddr = reverseDots(clientIP)
		} else {
			remoteAddr = clientIP
		}
	} else if forwardedFor != "" {
		if strings.Contains(forwardedFor, ",") {
			ips := strings.Split(forwardedFor, ", ")
			for i := len(ips) - 1; i >= 0; i-- {
				ip := ips[i]
				if privateIPRe.MatchString(ip) && !privateIPRe.MatchString(remoteAddr) {
					continue
				}
				remoteAddr = strings.TrimSpace(ip)
				break
			}
		} else if !privateIPRe.MatchString(forwardedFor) || privateIPRe.MatchString(remoteAddr) {
			remoteAddr = forwardedFor
		}
	}

	// Some final checking.
	if !ipRe.MatchString(remoteAddr) {
		remoteAddr = ""
	}
	c.IP = remoteAddr
}

func firstToken(s string, sep byte) string {
	if i := strings.IndexByte(s, sep); i >= 0 {
		return s[:i]
	}
	return s
}

// lastChunk is PHP strrchr: the suffix starting at the last sep.
func lastChunk(s string, sep byte) string {
	if i := strings.LastIndexByte(s, sep); i >= 0 {
		return s[i:]
	}
	return ""
}

func reverseDots(s string) string {
	parts := strings.Split(s, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}

// splitBrackets splits "a[b][]" into ("a", ["b", ""]).
func splitBrackets(key string) (string, []string) {
	br := strings.IndexByte(key, '[')
	if br < 0 {
		return key, nil
	}
	base := key[:br]
	var path []string
	rest := key[br:]
	for len(rest) > 0 && rest[0] == '[' {
		end := strings.IndexByte(rest, ']')
		if end < 0 {
			path = append(path, rest[1:])
			break
		}
		path = append(path, rest[1:end])
		rest = rest[end+1:]
	}
	return base, path
}
