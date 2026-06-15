package app

// Port of Sources/ManageSmileys.php: the smiley/message-icon admin
// (?action=smileys) — settings, smiley-set management, add/edit/order smileys,
// and message-icon CRUD.
//
// Deviations:
//   - Smiley image uploads are not supported (the package/filesystem upload
//     path); the "use existing file" path — picking a filename already present
//     in the Smileys directory — is fully ported, which is the common case.
//   - InstallSmileySet downloads+extracts a .tgz via the (dropped) package
//     manager; it is a documented no-op that redirects back to the set list.
//   - sortSmileyTable (MySQL ALTER TABLE ... ORDER BY) is a no-op: smiley
//     parsing now orders by LENGTH(code) DESC in the query (see bbc_smileys.go).

import (
	"os"
	"sort"
	"strings"
)

func init() {
	registerAction("smileys", (*Ctx).ManageSmileys)
}

func (c *Ctx) ManageSmileys() {
	a := c.App
	c.isAllowedTo("manage_smileys")
	c.adminIndex("manage_smileys")
	c.loadLanguage("ManageSmileys")

	subActions := map[string]bool{
		"addsmiley": true, "editicon": true, "editicons": true, "editsets": true,
		"editsmileys": true, "import": true, "modifyset": true, "modifysmiley": true,
		"setorder": true, "settings": true, "install": true,
	}
	sa := c.REQUEST.Str("sa")
	if !subActions[sa] {
		sa = "settings"
	}
	c.PageTitle = c.Txt("smileys_manage")
	c.smileySubAction = sa

	scripturl := a.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("smileys_manage"), Help: "smileys", Description: c.Txt("smiley_settings_explain")}
	selectedTab := smileyTabFor(sa)
	add := func(key, title, desc string) {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: title, Description: desc, Href: scripturl + "?action=smileys;sa=" + key, IsSelected: selectedTab == key})
	}
	add("editsets", c.Txt("smiley_sets"), c.Txt("smiley_editsets_explain"))
	if !a.SettingEmpty("smiley_enable") {
		add("addsmiley", c.Txt("smileys_add"), c.Txt("smiley_addsmiley_explain"))
		add("editsmileys", c.Txt("smileys_edit"), c.Txt("smiley_editsmileys_explain"))
		add("setorder", c.Txt("smileys_set_order"), c.Txt("smiley_setorder_explain"))
	}
	if !a.SettingEmpty("messageIcons_enable") {
		add("editicons", c.Txt("icons_edit_message_icons"), c.Txt("icons_edit_icons_explain"))
	}
	add("settings", c.Txt("settings"), c.Txt("smiley_settings_explain"))
	if len(tabs.Tabs) > 0 {
		tabs.Tabs[len(tabs.Tabs)-1].IsLast = true
	}
	c.AdminTabs = tabs

	switch sa {
	case "addsmiley":
		c.AddSmiley()
	case "editicon", "editicons":
		c.EditMessageIcons()
	case "editsets", "import", "modifyset":
		c.EditSmileySets()
	case "editsmileys", "modifysmiley":
		c.EditSmileys()
	case "setorder":
		c.EditSmileyOrder()
	case "install":
		c.InstallSmileySet()
	default:
		c.EditSmileySettings()
	}
}

func smileyTabFor(sa string) string {
	switch sa {
	case "editsets", "modifyset", "import":
		return "editsets"
	case "modifysmiley", "editsmileys":
		return "editsmileys"
	case "editicon", "editicons":
		return "editicons"
	default:
		return sa
	}
}

// SmileySet is one entry of $context['smiley_sets'].
type SmileySet struct {
	ID       int
	Path     string
	Name     string
	Selected bool
}

// loadSmileySets builds $context['smiley_sets'] from the known/names settings.
func (c *Ctx) loadSmileySets() []SmileySet {
	a := c.App
	paths := strings.Split(a.Setting("smiley_sets_known"), ",")
	names := strings.Split(a.Setting("smiley_sets_names"), "\n")
	def := a.Setting("smiley_sets_default")
	sets := make([]SmileySet, len(paths))
	for i, p := range paths {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		sets[i] = SmileySet{ID: i, Path: Htmlspecialchars(p), Name: Htmlspecialchars(name), Selected: p == def}
	}
	return sets
}

func (c *Ctx) smileysDir() (string, bool) {
	a := c.App
	dir := a.Setting("smileys_dir")
	if dir == "" {
		dir = a.Setting("boarddir") + "/Smileys"
	}
	info, err := os.Stat(dir)
	return dir, err == nil && info.IsDir()
}

var smileyImageExts = map[string]bool{".jpg": true, ".gif": true, ".jpeg": true, ".png": true}

// SmileyFilename is one entry of $context['filenames'].
type SmileyFilename struct {
	ID       string
	Selected bool
}

// smileyFilenames scans every set directory for usable image files.
func (c *Ctx) smileyFilenames(sets []SmileySet, dir string, found bool) []SmileyFilename {
	if !found {
		return nil
	}
	byLower := map[string]string{}
	for _, set := range sets {
		entries, err := os.ReadDir(dir + "/" + unHtmlspecialchars(set.Path))
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			ext := strings.ToLower(name[strings.LastIndex(name, "."):])
			if dot := strings.LastIndex(name, "."); dot >= 0 && smileyImageExts[ext] {
				low := strings.ToLower(name)
				if _, ok := byLower[low]; !ok {
					byLower[low] = Htmlspecialchars(name)
				}
			}
		}
	}
	keys := make([]string, 0, len(byLower))
	for k := range byLower {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]SmileyFilename, 0, len(keys))
	for _, k := range keys {
		out = append(out, SmileyFilename{ID: byLower[k]})
	}
	return out
}

// invalidateSmileyCache mirrors the cache_put_data(null) calls.
func (a *App) invalidateSmileyCache() {
	a.cache.Put("parsing_smileys", nil, 0)
	a.cache.Put("posting_smileys", nil, 0)
}

// ---- EditSmileySettings ----

type smileySettingsPage struct {
	Sets            []SmileySet
	SmileysDir      string
	SmileysDirFound bool
}

func (c *Ctx) EditSmileySettings() {
	a := c.App

	if c.POST.Has("sc") && c.POST.Has("smiley_sets_url") {
		c.checkSession("post", "", true)
		sets := strings.Split(a.Setting("smiley_sets_known"), ",")
		if c.POST.Has("smiley_enable") {
			c.sortSmileyTable()
		}
		defIdx := c.POST.Int("default_smiley_set")
		defSet := "default"
		if defIdx >= 0 && defIdx < len(sets) && sets[defIdx] != "" {
			defSet = sets[defIdx]
		}
		a.UpdateSettings(map[string]string{
			"smiley_sets_default": defSet,
			"smiley_sets_enable":  boolSetting(c.POST.Has("smiley_sets_enable")),
			"smiley_enable":       boolSetting(c.POST.Has("smiley_enable")),
			"messageIcons_enable": boolSetting(c.POST.Has("messageIcons_enable")),
			"smileys_url":         c.POST.Str("smiley_sets_url"),
			"smileys_dir":         c.POST.Str("smiley_sets_dir"),
		})
		a.invalidateSmileyCache()
		c.redirectExit("action=smileys;sa=settings")
		return
	}

	dir, found := c.smileysDir()
	c.Page = &smileySettingsPage{Sets: c.loadSmileySets(), SmileysDir: dir, SmileysDirFound: found}
	c.SubTemplate = templateSmileySettings
}

func boolSetting(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// ---- EditSmileySets ----

// SmileySetCur is $context['current_set'] on the modify form.
type SmileySetCur struct {
	ID           int
	Path         string
	Name         string
	Selected     bool
	IsNew        bool
	FixedDefault bool // PHP id == 'default' (the index-0 set)
	CanImport    int
}

// SmileySetDir is $context['smiley_set_dirs'].
type SmileySetDir struct {
	ID         string
	Selectable bool
	Current    bool
}

type editSetsPage struct {
	Sets        []SmileySet
	SelectedSet string
	SmileysURL  string
}

type modifySetPage struct {
	Current      SmileySetCur
	SetDirs      []SmileySetDir
	SmileysURL   string
	SmileyEnable bool
}

func (c *Ctx) EditSmileySets() {
	a := c.App
	subAction := c.smileySubAction

	if c.POST.Has("sc") {
		c.checkSession("post", "", true)

		if c.POST.Has("delete") && c.POST.Arr("smiley_set") != nil {
			paths := strings.Split(a.Setting("smiley_sets_known"), ",")
			names := strings.Split(a.Setting("smiley_sets_names"), "\n")
			del := map[int]bool{}
			c.POST.Arr("smiley_set").Values(func(k string, v any) {
				id := atoi(k)
				if id != 0 {
					del[id] = true
				}
			})
			var newPaths, newNames []string
			for i := range paths {
				if del[i] {
					continue
				}
				newPaths = append(newPaths, paths[i])
				if i < len(names) {
					newNames = append(newNames, names[i])
				}
			}
			def := a.Setting("smiley_sets_default")
			if !inStrings(newPaths, def) && len(newPaths) > 0 {
				def = newPaths[0]
			}
			a.UpdateSettings(map[string]string{
				"smiley_sets_known":   strings.Join(newPaths, ","),
				"smiley_sets_names":   strings.Join(newNames, "\n"),
				"smiley_sets_default": def,
			})
			a.invalidateSmileyCache()
		} else if c.POST.Has("add") {
			subAction = "modifyset"
		} else if c.POST.Has("set") {
			paths := strings.Split(a.Setting("smiley_sets_known"), ",")
			names := strings.Split(a.Setting("smiley_sets_names"), "\n")
			set := c.POST.Int("set")
			newPath := c.POST.Str("smiley_sets_path")
			newName := c.POST.Str("smiley_sets_name")

			if c.POST.Str("set") == "-1" && c.POST.Has("smiley_sets_path") {
				if inStrings(paths, newPath) {
					c.fatalLangError("smiley_set_already_exists", true)
				}
				def := a.Setting("smiley_sets_default")
				if c.POST.Has("smiley_sets_default") {
					def = newPath
				}
				a.UpdateSettings(map[string]string{
					"smiley_sets_known":   a.Setting("smiley_sets_known") + "," + newPath,
					"smiley_sets_names":   a.Setting("smiley_sets_names") + "\n" + newName,
					"smiley_sets_default": def,
				})
			} else {
				if set < 0 || set >= len(paths) || set >= len(names) {
					c.fatalLangError("smiley_set_not_found", true)
				}
				if inStrings(paths, newPath) && newPath != paths[set] {
					c.fatalLangError("smiley_set_path_already_used", true)
				}
				paths[set] = newPath
				names[set] = newName
				def := a.Setting("smiley_sets_default")
				if c.POST.Has("smiley_sets_default") {
					def = newPath
				}
				a.UpdateSettings(map[string]string{
					"smiley_sets_known":   strings.Join(paths, ","),
					"smiley_sets_names":   strings.Join(names, "\n"),
					"smiley_sets_default": def,
				})
			}
			if c.POST.Has("smiley_sets_import") {
				c.importSmileys(newPath)
			}
			a.invalidateSmileyCache()
		}
	}

	sets := c.loadSmileySets()

	// Import from an existing set directory.
	if subAction == "import" {
		c.checkSession("get", "", true)
		setIdx := c.GET.Int("set")
		if setIdx >= 0 && setIdx < len(sets) {
			c.importSmileys(unHtmlspecialchars(sets[setIdx].Path))
		}
		subAction = "modifyset"
	}

	if subAction == "modifyset" {
		c.smileyModifySet(sets)
		return
	}

	c.Page = &editSetsPage{Sets: sets, SelectedSet: a.Setting("smiley_sets_default"), SmileysURL: a.Setting("smileys_url")}
	c.SubTemplate = templateEditsets
}

func (c *Ctx) smileyModifySet(sets []SmileySet) {
	a := c.App
	setIdx := -1
	if c.GET.Has("set") {
		setIdx = c.GET.Int("set")
	}

	page := &modifySetPage{SmileysURL: a.Setting("smileys_url"), SmileyEnable: !a.SettingEmpty("smiley_enable")}

	dir := a.Setting("smileys_dir")
	if setIdx == -1 || setIdx < 0 || setIdx >= len(sets) {
		page.Current = SmileySetCur{ID: -1, IsNew: true}
	} else {
		s := sets[setIdx]
		page.Current = SmileySetCur{ID: s.ID, Path: s.Path, Name: s.Name, Selected: s.Selected, IsNew: false, FixedDefault: s.ID == 0}
		// Count importable smileys in this set's directory.
		if !a.SettingEmpty("smiley_enable") && dir != "" {
			if entries, err := os.ReadDir(dir + "/" + s.Path); err == nil {
				files := map[string]string{}
				for _, e := range entries {
					name := e.Name()
					if dot := strings.LastIndex(name, "."); dot >= 0 && smileyImageExts[strings.ToLower(name[dot:])] {
						files[strings.ToLower(name)] = name
					}
				}
				c.removeKnownSmileyFiles(files)
				page.Current.CanImport = len(files)
			}
		}
	}

	// Potential set directories.
	if dir != "" {
		if entries, err := os.ReadDir(dir); err == nil {
			known := strings.Split(a.Setting("smiley_sets_known"), ",")
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				name := e.Name()
				page.SetDirs = append(page.SetDirs, SmileySetDir{
					ID:         name,
					Selectable: name == page.Current.Path || !inStrings(known, name),
					Current:    name == page.Current.Path,
				})
			}
		}
	}

	c.Page = page
	c.SubTemplate = templateModifyset
}

// removeKnownSmileyFiles drops files already present in the smileys table.
func (c *Ctx) removeKnownSmileyFiles(files map[string]string) {
	a := c.App
	if len(files) == 0 {
		return
	}
	rows, err := a.DB.Query(a.Q(`SELECT filename FROM {$db_prefix}smileys`))
	if err != nil {
		return
	}
	for rows.Next() {
		var fn string
		rows.Scan(&fn)
		delete(files, strings.ToLower(fn))
	}
	rows.Close()
}

// importSmileys is ImportSmileys($smileyPath).
func (c *Ctx) importSmileys(smileyPath string) {
	a := c.App
	dir := a.Setting("smileys_dir")
	info, err := os.Stat(dir + "/" + smileyPath)
	if dir == "" || err != nil || !info.IsDir() {
		c.fatalLangError("smiley_set_unable_to_import", true)
	}
	entries, _ := os.ReadDir(dir + "/" + smileyPath)
	files := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if dot := strings.LastIndex(name, "."); dot >= 0 && smileyImageExts[strings.ToLower(name[dot:])] {
			files[strings.ToLower(name)] = name
		}
	}
	c.removeKnownSmileyFiles(files)
	if len(files) == 0 {
		return
	}

	var maxOrder int
	a.DB.QueryRow(a.Q(`SELECT IFNULL(MAX(smileyOrder), 0) FROM {$db_prefix}smileys WHERE hidden = 0 AND smileyRow = 0`)).Scan(&maxOrder)

	// Deterministic order by filename.
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fn := files[k]
		if len(fn) > 48 {
			continue
		}
		base := fn
		if dot := strings.Index(fn, "."); dot >= 0 {
			base = fn[:dot]
		}
		maxOrder++
		a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}smileys (code, filename, description, smileyRow, smileyOrder)
			VALUES (substr(?, 1, 30), ?, substr(?, 1, 80), 0, ?)`),
			":"+base+":", fn, base, maxOrder)
	}
	c.sortSmileyTable()
	a.invalidateSmileyCache()
}

// ---- AddSmiley ----

// CurrentSmiley is $context['current_smiley'].
type CurrentSmiley struct {
	ID          int
	Code        string
	Filename    string
	Description string
	Location    int
	IsNew       bool
}

type addSmileyPage struct {
	Sets        []SmileySet
	SelectedSet string
	Filenames   []SmileyFilename
	Current     CurrentSmiley
	SmileysURL  string
}

func (c *Ctx) AddSmiley() {
	a := c.App
	dir, found := c.smileysDir()
	sets := c.loadSmileySets()

	if c.POST.Has("sc") && c.POST.Has("smiley_code") {
		c.checkSession("post", "", true)
		code := Htmltrim(c.POST.Str("smiley_code"))
		location := c.POST.Int("smiley_location")
		if location > 2 || location < 0 {
			location = 0
		}
		filename := Htmltrim(c.POST.Str("smiley_filename"))

		if code == "" {
			c.fatalLangError("smiley_has_no_code", true)
		}
		var dup int
		a.DB.QueryRow(a.Q(`SELECT ID_SMILEY FROM {$db_prefix}smileys WHERE code = ?`), code).Scan(&dup)
		if dup > 0 {
			c.fatalLangError("smiley_not_unique", true)
		}
		// Upload methods are not supported in this port (see file header).
		if c.POST.Str("method") != "existing" {
			c.fatalError(c.Txt("smileys_upload_error"), false)
		}
		if filename == "" {
			c.fatalLangError("smiley_has_no_filename", true)
		}

		smileyOrder := 0
		if location != 1 {
			a.DB.QueryRow(a.Q(`SELECT IFNULL(MAX(smileyOrder) + 1, 0) FROM {$db_prefix}smileys WHERE hidden = ? AND smileyRow = 0`), location).Scan(&smileyOrder)
		}
		a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}smileys (code, filename, description, hidden, smileyOrder)
			VALUES (substr(?, 1, 30), substr(?, 1, 48), substr(?, 1, 80), ?, ?)`),
			code, filename, c.POST.Str("smiley_description"), location, smileyOrder)
		a.invalidateSmileyCache()
		c.redirectExit("action=smileys;sa=editsmileys")
		return
	}

	filenames := c.smileyFilenames(sets, dir, found)
	firstFile := ""
	if len(filenames) > 0 {
		firstFile = filenames[0].ID
	}
	c.Page = &addSmileyPage{
		Sets:        sets,
		SelectedSet: a.Setting("smiley_sets_default"),
		Filenames:   filenames,
		SmileysURL:  a.Setting("smileys_url"),
		Current: CurrentSmiley{
			Filename:    firstFile,
			Description: c.Txt("smileys_default_description"),
			IsNew:       true,
		},
	}
	c.SubTemplate = templateAddsmiley
}

// ---- EditSmileys / modifysmiley ----

// SmileyRow is one row of $context['smileys'] on the edit list.
type SmileyRow struct {
	ID           int
	Code         string
	Filename     string
	Description  string
	Location     string
	SetsNotFound []string
}

type editSmileysPage struct {
	Sets        []SmileySet
	SelectedSet string
	Sort        string
	Smileys     []SmileyRow
	SmileysURL  string
}

type modifySmileyPage struct {
	Sets        []SmileySet
	SelectedSet string
	Filenames   []SmileyFilename
	Current     CurrentSmiley
	SmileysURL  string
}

var smileySortCols = map[string]bool{"code": true, "filename": true, "description": true, "hidden": true}

func (c *Ctx) EditSmileys() {
	a := c.App

	if c.POST.Has("sc") {
		c.checkSession("post", "", true)

		if c.POST.Has("smiley_action") && c.POST.Arr("checked_smileys") != nil {
			var ids []int
			c.POST.Arr("checked_smileys").Values(func(k string, v any) {
				s, _ := v.(string)
				ids = append(ids, atoi(s))
			})
			if len(ids) > 0 {
				action := c.POST.Str("smiley_action")
				if action == "delete" {
					a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}smileys WHERE ID_SMILEY IN (` + joinInts(ids) + `)`))
				} else {
					displayTypes := map[string]int{"post": 0, "hidden": 1, "popup": 2}
					if h, ok := displayTypes[action]; ok {
						a.DB.Exec(a.Q(`UPDATE {$db_prefix}smileys SET hidden = ? WHERE ID_SMILEY IN (`+joinInts(ids)+`)`), h)
					}
				}
			}
		} else if c.POST.Has("smiley") {
			smiley := c.POST.Int("smiley")
			code := Htmltrim(c.POST.Str("smiley_code"))
			filename := Htmltrim(c.POST.Str("smiley_filename"))
			location := c.POST.Int("smiley_location")
			if location > 2 || location < 0 {
				location = 0
			}
			if code == "" {
				c.fatalLangError("smiley_has_no_code", true)
			}
			if filename == "" {
				c.fatalLangError("smiley_has_no_filename", true)
			}
			var dup int
			if smiley == 0 {
				a.DB.QueryRow(a.Q(`SELECT ID_SMILEY FROM {$db_prefix}smileys WHERE code = ?`), code).Scan(&dup)
			} else {
				a.DB.QueryRow(a.Q(`SELECT ID_SMILEY FROM {$db_prefix}smileys WHERE code = ? AND ID_SMILEY != ?`), code, smiley).Scan(&dup)
			}
			if dup > 0 {
				c.fatalLangError("smiley_not_unique", true)
			}
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}smileys SET code = ?, filename = ?, description = ?, hidden = ?
				WHERE ID_SMILEY = ?`),
				code, filename, c.POST.Str("smiley_description"), location, smiley)
			c.sortSmileyTable()
		}
		a.invalidateSmileyCache()
	}

	sets := c.loadSmileySets()

	if c.smileySubAction == "modifysmiley" {
		c.smileyModify(sets)
		return
	}

	// editsmileys list.
	sortCol := c.REQUEST.Str("sort")
	if !smileySortCols[sortCol] {
		sortCol = "filename"
	}
	page := &editSmileysPage{Sets: sets, SelectedSet: a.Setting("smiley_sets_default"), Sort: sortCol, SmileysURL: a.Setting("smileys_url")}

	rows, err := a.DB.Query(a.Q(`SELECT ID_SMILEY, code, filename, description, hidden FROM {$db_prefix}smileys ORDER BY ` + sortCol))
	if err == nil {
		for rows.Next() {
			var id, hidden int
			var code, filename, description string
			rows.Scan(&id, &code, &filename, &description, &hidden)
			loc := c.Txt("smileys_location_form")
			if hidden == 1 {
				loc = c.Txt("smileys_location_hidden")
			} else if hidden == 2 {
				loc = c.Txt("smileys_location_popup")
			}
			page.Smileys = append(page.Smileys, SmileyRow{
				ID: id, Code: Htmlspecialchars(code), Filename: Htmlspecialchars(filename),
				Description: Htmlspecialchars(description), Location: loc,
			})
		}
		rows.Close()
	}

	// Flag smileys whose file is missing in some sets.
	dir := a.Setting("smileys_dir")
	if dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			for _, set := range sets {
				for i := range page.Smileys {
					if _, err := os.Stat(dir + "/" + unHtmlspecialchars(set.Path) + "/" + unHtmlspecialchars(page.Smileys[i].Filename)); err != nil {
						page.Smileys[i].SetsNotFound = append(page.Smileys[i].SetsNotFound, set.Path)
					}
				}
			}
		}
	}

	c.Page = page
	c.SubTemplate = templateEditsmileys
}

func (c *Ctx) smileyModify(sets []SmileySet) {
	a := c.App
	dir, found := c.smileysDir()
	filenames := c.smileyFilenames(sets, dir, found)

	var cur CurrentSmiley
	var code, filename, description string
	var hidden int
	err := a.DB.QueryRow(a.Q(`SELECT ID_SMILEY, code, filename, description, hidden FROM {$db_prefix}smileys WHERE ID_SMILEY = ?`), c.REQUEST.Int("smiley")).
		Scan(&cur.ID, &code, &filename, &description, &hidden)
	if err != nil {
		c.fatalLangError("smiley_not_found", true)
	}
	cur.Code = Htmlspecialchars(code)
	cur.Filename = Htmlspecialchars(filename)
	cur.Description = Htmlspecialchars(description)
	cur.Location = hidden

	for i := range filenames {
		if strings.EqualFold(filenames[i].ID, cur.Filename) {
			filenames[i].Selected = true
		}
	}

	c.Page = &modifySmileyPage{Sets: sets, SelectedSet: a.Setting("smiley_sets_default"), Filenames: filenames, Current: cur, SmileysURL: a.Setting("smileys_url")}
	c.SubTemplate = templateModifysmiley
}

// ---- EditSmileyOrder ----

// SmileyOrderItem is one smiley on the order page.
type SmileyOrderItem struct {
	ID          int
	Code        string
	Filename    string
	Description string
	Row         int
	Order       int
	Selected    bool
}

// SmileyOrderLoc is a postform/popup grouping on the order page.
type SmileyOrderLoc struct {
	ID          string
	Title       string
	Description string
	LastRow     int
	Rows        [][]SmileyOrderItem
}

type setOrderPage struct {
	Locations        []SmileyOrderLoc
	MoveSmiley       int
	SmileysURL       string
	SmileySetDefault string
}

func (c *Ctx) EditSmileyOrder() {
	a := c.App

	if c.GET.Has("sesc") {
		c.checkSession("get", "", true)
		location := 0
		if c.GET.Str("location") == "popup" {
			location = 2
		}
		source := c.GET.Int("source")
		if source == 0 {
			c.fatalLangError("smiley_not_found", true)
		}
		var smileyRow, smileyOrder, smileyLocation int
		if c.GET.Has("after") && c.GET.Int("after") != 0 {
			after := c.GET.Int("after")
			err := a.DB.QueryRow(a.Q(`SELECT smileyRow, smileyOrder, hidden FROM {$db_prefix}smileys WHERE hidden = ? AND ID_SMILEY = ?`), location, after).
				Scan(&smileyRow, &smileyOrder, &smileyLocation)
			if err != nil {
				c.fatalLangError("smiley_not_found", true)
			}
		} else {
			smileyRow = c.GET.Int("row")
			smileyOrder = -1
			smileyLocation = location
		}
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}smileys SET smileyOrder = smileyOrder + 1 WHERE hidden = ? AND smileyRow = ? AND smileyOrder > ?`), location, smileyRow, smileyOrder)
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}smileys SET smileyOrder = ?, smileyRow = ?, hidden = ? WHERE ID_SMILEY = ?`), smileyOrder+1, smileyRow, smileyLocation, source)
		a.invalidateSmileyCache()
	}

	move := c.REQUEST.Int("move")
	page := &setOrderPage{MoveSmiley: move, SmileysURL: a.Setting("smileys_url"), SmileySetDefault: a.Setting("smiley_sets_default")}

	// rows grouped by location -> smileyRow.
	type grp struct {
		rowsByRow map[int][]SmileyOrderItem
		rowOrder  []int
	}
	groups := map[string]*grp{
		"postform": {rowsByRow: map[int][]SmileyOrderItem{}},
		"popup":    {rowsByRow: map[int][]SmileyOrderItem{}},
	}
	rows, err := a.DB.Query(a.Q(`SELECT ID_SMILEY, code, filename, description, smileyRow, smileyOrder, hidden FROM {$db_prefix}smileys WHERE hidden != 1 ORDER BY smileyOrder, smileyRow`))
	if err == nil {
		for rows.Next() {
			var id, sr, so, hidden int
			var code, filename, description string
			rows.Scan(&id, &code, &filename, &description, &sr, &so, &hidden)
			loc := "postform"
			if hidden != 0 {
				loc = "popup"
			}
			g := groups[loc]
			if _, ok := g.rowsByRow[sr]; !ok {
				g.rowOrder = append(g.rowOrder, sr)
			}
			g.rowsByRow[sr] = append(g.rowsByRow[sr], SmileyOrderItem{
				ID: id, Code: Htmlspecialchars(code), Filename: Htmlspecialchars(filename),
				Description: Htmlspecialchars(description), Row: sr, Order: so, Selected: move != 0 && move == id,
			})
		}
		rows.Close()
	}

	for _, locName := range []string{"postform", "popup"} {
		g := groups[locName]
		loc := SmileyOrderLoc{ID: locName, LastRow: len(g.rowOrder)}
		if locName == "postform" {
			loc.Title = c.Txt("smileys_location_form")
			loc.Description = c.Txt("smileys_location_form_description")
		} else {
			loc.Title = c.Txt("smileys_location_popup")
			loc.Description = c.Txt("smileys_location_popup_description")
		}
		for _, sr := range g.rowOrder {
			loc.Rows = append(loc.Rows, g.rowsByRow[sr])
		}

		// Fix non-sequential rows/orders in the DB.
		hiddenVal := 0
		if locName == "popup" {
			hiddenVal = 2
		}
		for newRow, smileyRow := range loc.Rows {
			if len(smileyRow) == 0 {
				continue
			}
			if newRow != smileyRow[0].Row {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}smileys SET smileyRow = ? WHERE smileyRow = ? AND hidden = ?`), newRow, smileyRow[0].Row, hiddenVal)
				loc.Rows[newRow][0].Row = newRow
			}
			for orderID, smiley := range smileyRow {
				if orderID != smiley.Order {
					a.DB.Exec(a.Q(`UPDATE {$db_prefix}smileys SET smileyOrder = ? WHERE ID_SMILEY = ?`), orderID, smiley.ID)
				}
			}
		}
		page.Locations = append(page.Locations, loc)
	}

	a.invalidateSmileyCache()
	c.Page = page
	c.SubTemplate = templateSetorder
}

// ---- EditMessageIcons ----

// MessageIcon is one entry of $context['icons'].
type MessageIcon struct {
	ID        int
	Title     string
	Filename  string
	ImageURL  string
	BoardID   int
	Board     string
	Order     int
	TrueOrder int
	After     int
}

type editIconsPage struct {
	Icons []MessageIcon
}

// BoardOpt is one board option on the icon form.
type BoardOpt struct {
	ID   int
	Name string
}

type editIconPage struct {
	NewIcon bool
	Icon    MessageIcon
	Boards  []BoardOpt
	Icons   []MessageIcon
}

func (c *Ctx) EditMessageIcons() {
	a := c.App

	// Load all icons (ordered by iconOrder, since we can't physically reorder).
	icons := c.loadMessageIcons()
	iconByID := map[int]int{} // id -> index
	for i, ic := range icons {
		iconByID[ic.ID] = i
	}

	if c.POST.Has("sc") {
		c.checkSession("post", "", true)

		if c.POST.Has("delete") && c.POST.Arr("checked_icons") != nil {
			var ids []int
			c.POST.Arr("checked_icons").Values(func(k string, v any) {
				s, _ := v.(string)
				ids = append(ids, atoi(s))
			})
			if len(ids) > 0 {
				a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}message_icons WHERE ID_ICON IN (` + joinInts(ids) + `)`))
			}
		} else if c.smileySubAction == "editicon" && c.GET.Has("icon") {
			c.saveMessageIcon(icons, iconByID)
		}

		if !c.POST.Has("add") {
			c.redirectExit("action=smileys;sa=editicons")
			return
		}
	}

	// Add/edit icon form.
	if c.smileySubAction == "editicon" || c.POST.Has("add") {
		page := &editIconPage{Icons: icons}
		iconID := c.GET.Int("icon")
		idx, exists := iconByID[iconID]
		page.NewIcon = !c.GET.Has("icon") || !exists
		if !page.NewIcon {
			page.Icon = icons[idx]
		}
		rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name FROM {$db_prefix}boards WHERE ` + c.User.QuerySeeBoard))
		if err == nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				page.Boards = append(page.Boards, BoardOpt{ID: id, Name: name})
			}
			rows.Close()
		}
		c.Page = page
		c.SubTemplate = templateEditicon
		return
	}

	c.Page = &editIconsPage{Icons: icons}
	c.SubTemplate = templateEditicons
}

func (c *Ctx) loadMessageIcons() []MessageIcon {
	a := c.App
	var icons []MessageIcon
	rows, err := a.DB.Query(a.Q(`
		SELECT m.ID_ICON, m.title, m.filename, m.iconOrder, m.ID_BOARD, IFNULL(b.name, '') AS boardName
		FROM {$db_prefix}message_icons AS m
			LEFT JOIN {$db_prefix}boards AS b ON (b.ID_BOARD = m.ID_BOARD)
		WHERE ` + c.User.QuerySeeBoard + `
		ORDER BY m.iconOrder`))
	if err != nil {
		return nil
	}
	lastIcon, trueOrder := 0, 0
	imagesURL := c.Theme.Get("actual_images_url")
	defImagesURL := c.Theme.Get("default_images_url")
	themeDir := c.Theme.Get("theme_dir")
	for rows.Next() {
		var ic MessageIcon
		var boardName string
		rows.Scan(&ic.ID, &ic.Title, &ic.Filename, &ic.Order, &ic.BoardID, &boardName)
		url := defImagesURL
		if _, err := os.Stat(themeDir + "/images/post/" + ic.Filename + ".gif"); err == nil {
			url = imagesURL
		}
		ic.ImageURL = url + "/post/" + ic.Filename + ".gif"
		if boardName == "" {
			ic.Board = c.Txt("icons_edit_icons_all_boards")
		} else {
			ic.Board = boardName
		}
		ic.TrueOrder = trueOrder
		trueOrder++
		ic.After = lastIcon
		lastIcon = ic.ID
		icons = append(icons, ic)
	}
	rows.Close()
	return icons
}

func (c *Ctx) saveMessageIcon(icons []MessageIcon, iconByID map[int]int) {
	a := c.App
	iconID := c.GET.Int("icon")
	filename := c.POST.Str("icon_filename")
	if strings.Contains(filename, ".gif") {
		filename = strings.TrimSuffix(filename, ".gif")
	}
	if _, err := os.Stat(c.Theme.Get("default_theme_dir") + "/images/post/" + filename + ".gif"); err != nil {
		c.fatalLangError("icon_not_found", true)
	}
	if len(filename) > 16 {
		c.fatalLangError("icon_name_too_long", true)
	}
	iconLocation := c.POST.Int("icon_location")
	if iconLocation == iconID && iconID != 0 {
		c.fatalLangError("icon_after_itself", true)
	}

	// Work on a mutable copy of the true_order values.
	type ent struct {
		id, board, order int
		title, filename  string
	}
	list := make([]ent, len(icons))
	for i, ic := range icons {
		list[i] = ent{id: ic.ID, board: ic.BoardID, order: ic.TrueOrder, title: ic.Title, filename: ic.Filename}
	}
	idxByID := map[int]int{}
	for i, e := range list {
		idxByID[e.id] = i
	}

	// If editing, reduce the order of everything after it by one.
	if iconID != 0 {
		oldOrder := list[idxByID[iconID]].order
		for i := range list {
			if list[i].order > oldOrder {
				list[i].order--
			}
		}
	}
	// New order position.
	newOrder := 0
	if iconLocation != 0 {
		if idx, ok := idxByID[iconLocation]; ok {
			newOrder = list[idx].order + 1
		}
	}
	for i := range list {
		if list[i].order >= newOrder {
			list[i].order++
		}
	}

	// Set/insert the current icon.
	if idx, ok := idxByID[iconID]; ok && iconID != 0 {
		list[idx].order = newOrder
		list[idx].title = c.POST.Str("icon_description")
		list[idx].filename = filename
		list[idx].board = c.POST.Int("icon_board")
	} else {
		list = append(list, ent{id: 0, board: c.POST.Int("icon_board"), order: newOrder, title: c.POST.Str("icon_description"), filename: filename})
	}

	// REPLACE INTO for each (INSERT OR REPLACE). id 0 -> autoincrement insert.
	for _, e := range list {
		if e.id == 0 {
			a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}message_icons (ID_BOARD, title, filename, iconOrder) VALUES (?, substr(?, 1, 80), substr(?, 1, 80), ?)`),
				e.board, e.title, e.filename, e.order)
		} else {
			a.DB.Exec(a.Q(`INSERT OR REPLACE INTO {$db_prefix}message_icons (ID_ICON, ID_BOARD, title, filename, iconOrder) VALUES (?, ?, substr(?, 1, 80), substr(?, 1, 80), ?)`),
				e.id, e.board, e.title, e.filename, e.order)
		}
	}
}

// ---- InstallSmileySet / sortSmileyTable ----

// InstallSmileySet would download+extract a smiley-set package; the package
// manager is dropped in this port, so it is a no-op that returns to the list.
func (c *Ctx) InstallSmileySet() {
	c.checkSession("request", "", true)
	c.redirectExit("action=smileys")
}

// sortSmileyTable is a no-op: parse ordering is handled in the query (see the
// file header). Kept so the call sites read like the PHP.
func (c *Ctx) sortSmileyTable() {}
