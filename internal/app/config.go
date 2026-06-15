package app

// Config replaces Settings.php. Keys keep the Settings.php variable names so
// SMF documentation still applies; database credentials are replaced by a
// single SQLite path, and listen/assetdir are new (the web server is built in).

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// ########## Maintenance ##########
	Maintenance int    // 0 = off, 1 = maintenance mode, 2 = untouchable
	MTitle      string // title for the maintenance message
	MMessage    string // description of why the forum is down

	// ########## Forum Info ##########
	MbName         string // the name of the forum
	Language       string // default language file set
	BoardURL       string // URL to the forum, without trailing /
	WebmasterEmail string // address to send emails from
	CookieName     string // name of the login cookie

	// ########## Database Info ##########
	DBPath   string // path to the SQLite database file
	DBPrefix string // table prefix, e.g. "smf_"

	// ########## Server ##########
	Listen   string // address for the HTTP listener, e.g. ":8080"
	AssetDir string // directory containing Themes/, Smileys/, avatars/
}

func DefaultConfig() Config {
	return Config{
		Maintenance:    0,
		MTitle:         "Maintenance Mode",
		MMessage:       "Okay faithful users...we're attempting to restore an older backup of the database...news will be posted once we're back!",
		MbName:         "My Community",
		Language:       "english",
		BoardURL:       "http://127.0.0.1:8080",
		WebmasterEmail: "noreply@myserver.com",
		CookieName:     "SMFCookie11",
		DBPath:         "./smf.sqlite",
		DBPrefix:       "smf_",
		Listen:         ":8080",
		AssetDir:       ".",
	}
}

// LoadConfig reads a flat "key = value" file. Lines starting with # are
// comments. Unknown keys are an error so typos don't silently use defaults.
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	for lineNo, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return cfg, fmt.Errorf("%s:%d: expected key = value", path, lineNo+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "maintenance":
			cfg.Maintenance, err = strconv.Atoi(value)
			if err != nil {
				return cfg, fmt.Errorf("%s:%d: maintenance must be 0, 1 or 2", path, lineNo+1)
			}
		case "mtitle":
			cfg.MTitle = value
		case "mmessage":
			cfg.MMessage = value
		case "mbname":
			cfg.MbName = value
		case "language":
			cfg.Language = value
		case "boardurl":
			cfg.BoardURL = strings.TrimRight(value, "/")
		case "webmaster_email":
			cfg.WebmasterEmail = value
		case "cookiename":
			cfg.CookieName = value
		case "db_path":
			cfg.DBPath = value
		case "db_prefix":
			cfg.DBPrefix = value
		case "listen":
			cfg.Listen = value
		case "assetdir":
			cfg.AssetDir = value
		default:
			return cfg, fmt.Errorf("%s:%d: unknown setting %q", path, lineNo+1, key)
		}
	}
	return cfg, nil
}

// WriteDefaultConfig writes a commented sample config, used by `smf init`.
func WriteDefaultConfig(path string) error {
	const sample = `########## Maintenance ##########
# Set to 1 to enable Maintenance Mode, 2 to make the forum untouchable.
maintenance = 0
mtitle = Maintenance Mode
mmessage = Okay faithful users...we're attempting to restore an older backup of the database...news will be posted once we're back!

########## Forum Info ##########
mbname = My Community
language = english
# URL to your forum, without the trailing /!
boardurl = http://127.0.0.1:8080
webmaster_email = noreply@myserver.com
cookiename = SMFCookie11

########## Database ##########
db_path = ./smf.sqlite
db_prefix = smf_

########## Server ##########
listen = :8080
# Directory containing Themes/, Smileys/, avatars/ and the attachments dir.
assetdir = .
`
	return os.WriteFile(path, []byte(sample), 0644)
}
