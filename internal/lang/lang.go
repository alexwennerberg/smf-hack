// Package lang holds the $txt language strings from
// Themes/default/languages/*.english.php, converted to Go data by
// tools/txtgen (see english_gen.go).
//
// Static entries are plain strings. Dynamic entries — PHP lines like
//
//	$txt[18] = $context['forum_name'] . ' - Index';
//
// are stored as templates with {placeholder} markers that the app resolves
// per request: {forum_name}, {scripturl}, {boardurl}, {forum_version},
// {forum_copyright}, {webmaster_email}, {modSettings:key}, {txt:key}, and
// the one-off {password_min_chars}. Lists are PHP array entries (days,
// months); Base records the starting index (months are 1-based).
package lang

type List struct {
	Base  int
	Items []string
}

type File struct {
	Strings map[string]string
	Dynamic map[string]string
	Lists   map[string]List
	// HelpStrings/HelpDynamic hold $helptxt entries, kept in their own
	// namespace: SMF's $txt and $helptxt are distinct arrays that share many
	// keys (e.g. "enableSpellChecking" is a short label in $txt and a long
	// description in $helptxt). Merging them would let one clobber the other.
	HelpStrings map[string]string
	HelpDynamic map[string]string
}
