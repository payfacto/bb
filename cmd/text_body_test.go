package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTextBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.md")
	if err := os.WriteFile(path, []byte("file body\nwith newlines"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		inline  string
		file    string
		want    string
		wantErr bool
	}{
		{name: "neither", inline: "", file: "", want: ""},
		{name: "inline only", inline: "hi", file: "", want: "hi"},
		{name: "file only", inline: "", file: path, want: "file body\nwith newlines"},
		{name: "both set", inline: "x", file: path, wantErr: true},
		{name: "missing file", inline: "", file: filepath.Join(dir, "nope.md"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTextBody(tc.inline, tc.file, "description", "description-file")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
