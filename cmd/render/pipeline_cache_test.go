package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestPipelineCacheListString_empty(t *testing.T) {
	if got := render.PipelineCacheListString(nil); got != "No pipeline caches found.\n" {
		t.Errorf("unexpected empty output: %q", got)
	}
}

func TestPipelineCacheListString_rows(t *testing.T) {
	caches := []bitbucket.PipelineCache{{
		UUID:          "{cache-1}",
		Name:          "gradle",
		Path:          "~/.gradle/caches",
		FileSizeBytes: 3 * 1024 * 1024,
		CreatedOn:     "2026-11-01T10:00:00Z",
	}}
	out := render.PipelineCacheListString(caches)
	for _, want := range []string{"NAME", "SIZE", "gradle", "{cache-1}", "3.0 MiB", "2026-11-01", "~/.gradle/caches"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %q", want, out)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:                  "0 B",
		1023:               "1023 B",
		1024:               "1.0 KiB",
		1536:               "1.5 KiB",
		5 * 1024 * 1024:    "5.0 MiB",
		2 * 1024 * 1 << 30: "2.0 TiB",
	}
	for in, want := range cases {
		if got := render.HumanBytes(in); got != want {
			t.Errorf("HumanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
