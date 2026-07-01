package bitbucket

import "strings"

// bbqlQuote returns s as a BBQL double-quoted string literal, escaping any
// backslash and double-quote so user-supplied values cannot break out of the
// quoted string or inject additional query clauses. Bitbucket BBQL uses
// backslash escaping inside double-quoted strings (e.g. name = "a \" b").
// The backslash must be escaped before the quote so an input backslash does
// not consume the quote's escape.
func bbqlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
