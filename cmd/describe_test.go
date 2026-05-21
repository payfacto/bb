package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestEveryLeafIsRegistered fails when a leaf command in the live Cobra tree
// is missing from commandRegistry. This is the build-time guarantee that the
// --describe manifest is always complete.
func TestEveryLeafIsRegistered(t *testing.T) {
	var missing []string
	visit(rootCmd, nil, func(path []string, c *cobra.Command) {
		key := strings.Join(path, " ")
		if _, ok := commandRegistry[key]; !ok {
			missing = append(missing, key)
		}
	})
	if len(missing) > 0 {
		t.Fatalf("commands not in commandRegistry (cmd/describe.go): %v", missing)
	}
}

// TestRegistryReferencesOnlyRealCommands fails when commandRegistry has an
// entry for a path that no longer exists in the Cobra tree (stale entry).
func TestRegistryReferencesOnlyRealCommands(t *testing.T) {
	real := map[string]bool{}
	visit(rootCmd, nil, func(path []string, c *cobra.Command) {
		real[strings.Join(path, " ")] = true
	})
	var stale []string
	for key := range commandRegistry {
		if !real[key] {
			stale = append(stale, key)
		}
	}
	if len(stale) > 0 {
		t.Fatalf("commandRegistry has stale entries (no matching command): %v", stale)
	}
}

// TestEveryRegisteredTypeResolves fails when an OutputType or StdinType
// references a key that is not present in typeRegistry.
func TestEveryRegisteredTypeResolves(t *testing.T) {
	var missing []string
	for key, spec := range commandRegistry {
		if spec.OutputType != "" {
			if _, ok := typeRegistry[spec.OutputType]; !ok {
				missing = append(missing, spec.OutputType+" (output for "+key+")")
			}
		}
		if spec.StdinType != "" {
			if _, ok := typeRegistry[spec.StdinType]; !ok {
				missing = append(missing, spec.StdinType+" (stdin for "+key+")")
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("types not in typeRegistry (cmd/describe.go): %v", missing)
	}
}

// TestBuildManifestSucceeds is the end-to-end check: walks the live tree,
// resolves every type via reflection, and asserts the resulting manifest is
// non-empty and well-shaped.
func TestBuildManifestSucceeds(t *testing.T) {
	m, err := buildManifest(rootCmd, "test")
	if err != nil {
		t.Fatalf("buildManifest: %v", err)
	}
	if len(m.Commands) == 0 {
		t.Fatal("manifest contained zero commands")
	}
	for _, c := range m.Commands {
		if c.Action == "" {
			t.Errorf("command %v has empty action", c.Path)
		}
	}
}
