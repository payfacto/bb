package bitbucket

import (
	"strings"
	"testing"
)

func TestStripControlChars(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"plain text unchanged":       {"hello world", "hello world"},
		"newline and tab kept":       {"a\nb\tc", "a\nb\tc"},
		"ESC sequence neutralised":   {"\x1b[2Jcleared", "[2Jcleared"},
		"BEL and CR dropped":         {"ring\x07\rover", "ringover"},
		"DEL dropped":                {"a\x7fb", "ab"},
		"C1 CSI rune dropped":        {"a\u009b31mb", "a31mb"},
		"OSC hyperlink neutralised":  {"\x1b]8;;http://x\x07link", "]8;;http://xlink"},
		"non-ASCII letters kept":     {"café été", "café été"},
		"empty string":               {"", ""},
		"only control chars removed": {"\x00\x01\x1b", ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := stripControlChars(tc.in); got != tc.want {
				t.Errorf("stripControlChars(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDecode_StripsControlCharsFromNestedStrings(t *testing.T) {
	type inner struct {
		Name string `json:"name"`
	}
	type outer struct {
		Title  string            `json:"title"`
		Ptr    *inner            `json:"ptr"`
		List   []inner           `json:"list"`
		Labels map[string]string `json:"labels"`
		Any    any               `json:"any"`
	}
	data := []byte(`{
		"title": "t\u001b[31m",
		"ptr": {"name": "p\u0007"},
		"list": [{"name": "l\u001b]0;x"}],
		"labels": {"k\u001b": "v\u001b"},
		"any": {"nested": ["s\u001b", 1, {"deep": "d\u001b"}]}
	}`)
	got, err := decode[outer](data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "t[31m" || got.Ptr.Name != "p" || got.List[0].Name != "l]0;x" {
		t.Errorf("struct strings not cleaned: %+v", got)
	}
	if got.Labels["k"] != "v" {
		t.Errorf("map keys/values not cleaned: %#v", got.Labels)
	}
	nested := got.Any.(map[string]any)["nested"].([]any)
	if nested[0] != "s" || nested[2].(map[string]any)["deep"] != "d" {
		t.Errorf("interface values not cleaned: %#v", nested)
	}
}

func TestParseHTTPError_StripsControlCharsFromMessage(t *testing.T) {
	err := parseHTTPError(400, []byte(`{"error":{"message":"bad\u001b[2J"}}`))
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Message != "bad[2J" {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestParseHTTPError_StripsControlCharsFromRawBody(t *testing.T) {
	err := parseHTTPError(502, []byte("<html>bad gateway\x1b[2J\x07</html>"))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %#v", err)
	}
	if apiErr.Body != "<html>bad gateway[2J</html>" {
		t.Errorf("raw body not cleaned: %q", apiErr.Body)
	}
	if strings.ContainsAny(err.Error(), "\x1b\x07") {
		t.Errorf("Error() still contains control characters: %q", err.Error())
	}
}
