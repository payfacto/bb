package bitbucket

import (
	"reflect"
	"strings"
)

// Strings from the Bitbucket API (PR titles, comments, file paths, report
// text, ...) are written straight to a terminal by the text renderers and the
// TUI, and by users piping JSON through `jq -r`. Anyone who can author that
// content could embed escape sequences that clear the screen, rewrite earlier
// lines, or plant OSC hyperlinks. Every decoded response is therefore cleaned
// of control characters at the client boundary, before any caller sees it.
//
// Raw-content endpoints (pipeline step logs, PR diffs) are not decoded and are
// deliberately left untouched: their ANSI colours are legitimate output.

// c1ControlStart and c1ControlEnd bound the C1 control block (U+0080-U+009F),
// which includes the single-rune CSI (U+009B) and OSC (U+009D) introducers.
const (
	c1ControlStart = '\u0080'
	c1ControlEnd   = '\u009f'
	asciiDelete    = '\u007f'
	asciiSpace     = ' '
)

// stripControlChars removes C0 controls (except newline and tab), DEL, and C1
// controls from s. Printable text, including non-ASCII letters, is unchanged.
func stripControlChars(s string) string {
	if strings.IndexFunc(s, isUnsafeControl) < 0 {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isUnsafeControl(r) {
			return -1
		}
		return r
	}, s)
}

func isUnsafeControl(r rune) bool {
	switch {
	case r == '\n' || r == '\t':
		return false
	case r < asciiSpace || r == asciiDelete:
		return true
	default:
		return r >= c1ControlStart && r <= c1ControlEnd
	}
}

// stripControlCharsDeep cleans every string reachable from v (struct fields,
// pointers, slices, arrays, map keys and values, and interface values) in
// place. v must be a pointer so that the cleaned values can be stored.
func stripControlCharsDeep(v any) {
	cleanValue(reflect.ValueOf(v))
}

func cleanValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			cleanValue(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				cleanValue(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			cleanValue(v.Index(i))
		}
	case reflect.Map:
		cleanMap(v)
	case reflect.Interface:
		if !v.IsNil() && v.CanSet() {
			v.Set(cleanedCopy(v.Elem()))
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(stripControlChars(v.String()))
		}
	}
}

// cleanMap rebuilds a map's entries, since map elements are not addressable.
func cleanMap(m reflect.Value) {
	if m.IsNil() {
		return
	}
	cleaned := reflect.MakeMapWithSize(m.Type(), m.Len())
	iter := m.MapRange()
	for iter.Next() {
		cleaned.SetMapIndex(cleanedCopy(iter.Key()), cleanedCopy(iter.Value()))
	}
	m.Clear()
	iter = cleaned.MapRange()
	for iter.Next() {
		m.SetMapIndex(iter.Key(), iter.Value())
	}
}

// cleanedCopy returns an addressable, cleaned copy of v, for values (map
// entries, interface contents) that cannot be modified in place.
func cleanedCopy(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	cp := reflect.New(v.Type()).Elem()
	cp.Set(v)
	cleanValue(cp)
	return cp
}
