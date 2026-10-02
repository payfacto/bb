package cmd

import (
	"testing"

	"github.com/payfacto/bb/internal/config"
)

func TestParseAuthMethodChoice(t *testing.T) {
	tests := []struct {
		input  string
		def    string
		want   string
		wantOK bool
	}{
		{"", config.AuthTypeOAuth, config.AuthTypeOAuth, true},
		{"", config.AuthTypeAPIToken, config.AuthTypeAPIToken, true},
		{"1", config.AuthTypeAPIToken, config.AuthTypeOAuth, true},
		{"oauth", config.AuthTypeAPIToken, config.AuthTypeOAuth, true},
		{" OAuth \n", config.AuthTypeAPIToken, config.AuthTypeOAuth, true},
		{"2", config.AuthTypeOAuth, config.AuthTypeAPIToken, true},
		{"token", config.AuthTypeOAuth, config.AuthTypeAPIToken, true},
		{"api", config.AuthTypeOAuth, config.AuthTypeAPIToken, true},
		{"3", config.AuthTypeOAuth, "", false},
		{"banana", config.AuthTypeOAuth, "", false},
	}
	for _, tc := range tests {
		got, ok := parseAuthMethodChoice(tc.input, tc.def)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("parseAuthMethodChoice(%q, %q) = (%q, %v), want (%q, %v)",
				tc.input, tc.def, got, ok, tc.want, tc.wantOK)
		}
	}
}
