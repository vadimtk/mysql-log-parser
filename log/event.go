package log

import (
	"strings"
	"regexp"
)

var spaceRe *regexp.Regexp = regexp.MustCompile(`\s+`)
var nullRe *regexp.Regexp = regexp.MustCompile(`\bnull\b`)
var limitRe *regexp.Regexp = regexp.MustCompile(`\blimit \?(?:, ?\?| offset \?)?`)
var escapedQuoteRe *regexp.Regexp = regexp.MustCompile(`\\["']`)
var doubleQuotedValRe *regexp.Regexp = regexp.MustCompile(`".*?"`)
var singleQuotedValRe *regexp.Regexp = regexp.MustCompile(`'.*?'`)
var numberRe *regexp.Regexp = regexp.MustCompile(`\b[0-9+-][0-9a-f.xb+-]*`)
var valueListRe *regexp.Regexp = regexp.MustCompile(`\b(in|values?)(?:[\s,]*\([\s?,]*\))+`)
var multiLineCommentRe *regexp.Regexp = regexp.MustCompile(`(?sm)/\*[^!].*?\*/`)
// Go re doesn't support ?=, but I don't think slow logs can have -- comments,
// so we don't need this for now
//var oneLineCommentRe *regexp.Regexp = regexp.MustCompile(`(?:--|#)[^'"\r\n]*(?=[\r\n]|\z)`)
var useDbRe *regexp.Regexp = regexp.MustCompile(`\Ause .+\z`)
var unionRe *regexp.Regexp = regexp.MustCompile(`\b(select\s.*?)(?:(\sunion(?:\sall)?)\s$1)+`)

type Event struct {
	Offset uint64 // byte offset in log file, start of event
	Ts string     // if present in log file, often times not
	Admin bool    // Query is admin command not SQL query
	Query string  // SQL query or admin command
	User string
	Host string
	Db string
	TimeMetrics map[string]float32   // *_time and *_wait metrics
	NumberMetrics map[string]uint64  // most metrics
	BoolMetrics map[string]bool      // yes/no metrics
}

func NewEvent() *Event {
	event := new(Event)
	event.TimeMetrics = make(map[string]float32)
	event.NumberMetrics = make(map[string]uint64)
	event.BoolMetrics = make(map[string]bool)
	return event
}

func StripComments(q string) string {
	// @todo See comment above
	// q = oneLineCommentRe.ReplaceAllString(q, "")
	q = multiLineCommentRe.ReplaceAllString(q, "")
	return q
}

func QueryClass(q string) string {
	q = StripComments(q)
	q = strings.TrimSpace(q)
	q = spaceRe.ReplaceAllString(q, " ")

	if useDbRe.MatchString(q) {
		return "use ?"
	}

	q = escapedQuoteRe.ReplaceAllString(q, "")
	q = doubleQuotedValRe.ReplaceAllString(q, "?")
	q = singleQuotedValRe.ReplaceAllString(q, "?")
	q = numberRe.ReplaceAllString(q, "?")

	q = valueListRe.ReplaceAllString(q, "$1(?+)")
	q = unionRe.ReplaceAllString(q, "$1 /*repeat$2*/")

	q = strings.ToLower(q)

	// Must replace these after strings.ToLower().
	q = nullRe.ReplaceAllString(q, "?")
	q = limitRe.ReplaceAllString(q, "limit ?")

	return q
}

type EventDescription struct {
	Class string  // fingerprint of Query
	Id uint32     // CRC32 checksum of Class
	Alias string  // very short form of Query (distill)
}
