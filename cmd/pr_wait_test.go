package cmd

import (
	"testing"
	"time"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestApplySince(t *testing.T) {
	ts := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		since       string
		wantNewOnly bool
		wantAfterID int
		wantAfter   time.Time
		wantErr     bool
	}{
		{since: "", wantNewOnly: true},
		{since: "0"},
		{since: "123", wantAfterID: 123},
		{since: "2026-11-10T10:00:00Z", wantAfter: ts},
		{since: "-4", wantErr: true},
		{since: "yesterday", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.since, func(t *testing.T) {
			var opts bitbucket.WaitCommentsOptions
			err := applySince(&opts, tt.since)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if opts.NewOnly != tt.wantNewOnly || opts.Filter.AfterID != tt.wantAfterID || !opts.Filter.After.Equal(tt.wantAfter) {
				t.Errorf("got NewOnly=%v AfterID=%d After=%v", opts.NewOnly, opts.Filter.AfterID, opts.Filter.After)
			}
		})
	}
}

func TestCommentEventLine(t *testing.T) {
	c := bitbucket.Comment{ID: 9, User: bitbucket.Actor{DisplayName: "SYNAPSE PETRA"},
		Parent: &bitbucket.Parent{ID: 4}, Inline: &bitbucket.Inline{Path: "cmd/x.go"}}
	want := `comment=9 author="SYNAPSE PETRA" parent=4 path=cmd/x.go`
	if got := commentEventLine(c); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCommentsExitCode(t *testing.T) {
	if commentsExitCode(bitbucket.CommentsFound) != 0 || commentsExitCode(bitbucket.CommentsTimeout) != 3 {
		t.Error("expected found=0, timeout=3")
	}
}
