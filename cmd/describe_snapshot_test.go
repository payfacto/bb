package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden controls whether TestManifestSnapshot rewrites testdata files
// instead of asserting against them. Run with `go test ./cmd/ -update` to
// refresh after intentional manifest changes.
var updateGolden = flag.Bool("update", false, "update golden manifest file")

// TestManifestSnapshot locks the shape (not the values) of the manifest so
// accidental schema reshuffles surface in CI. We snapshot only the structure
// — keys, command paths, action classes, type names — not the full JSON
// Schemas (those would churn on every Bitbucket type tweak).
func TestManifestSnapshot(t *testing.T) {
	m, err := buildManifest(rootCmd, "snapshot")
	if err != nil {
		t.Fatalf("buildManifest: %v", err)
	}

	// Reduce to a stable shape: strip schemas + flag descriptions + summaries
	// (those churn with help-text edits and are not part of the contract we
	// want to lock).
	type flagShape struct {
		Name     string `json:"name"`
		Short    string `json:"short,omitempty"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
	}
	type stdinShape struct {
		Type     string `json:"type"`
		Behavior string `json:"behavior"`
	}
	type cmdShape struct {
		Path       []string    `json:"path"`
		Action     string      `json:"action"`
		Flags      []flagShape `json:"flags"`
		Stdin      *stdinShape `json:"stdin,omitempty"`
		OutputType string      `json:"output_type,omitempty"`
		Ordering   string      `json:"ordering,omitempty"`
	}
	type manifestShape struct {
		ManifestSchemaVersion string     `json:"manifest_schema_version"`
		Commands              []cmdShape `json:"commands"`
	}

	shape := manifestShape{
		ManifestSchemaVersion: m.ManifestSchemaVersion,
		Commands:              make([]cmdShape, 0, len(m.Commands)),
	}
	for _, c := range m.Commands {
		flags := make([]flagShape, 0, len(c.Flags))
		for _, f := range c.Flags {
			flags = append(flags, flagShape{
				Name:     f.Name,
				Short:    f.Short,
				Type:     f.Type,
				Required: f.Required,
			})
		}
		row := cmdShape{
			Path:       c.Path,
			Action:     c.Action,
			Flags:      flags,
			OutputType: c.OutputType,
			Ordering:   c.Ordering,
		}
		if c.Stdin != nil {
			row.Stdin = &stdinShape{Type: c.Stdin.Type, Behavior: c.Stdin.Behavior}
		}
		shape.Commands = append(shape.Commands, row)
	}

	got, err := json.MarshalIndent(shape, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')

	goldenPath := filepath.Join("testdata", "manifest.golden.json")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run `go test ./cmd/ -update` to generate)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("manifest shape drifted from testdata/manifest.golden.json — review diff and re-run with -update if intentional")
	}
}
