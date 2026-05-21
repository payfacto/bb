package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// manifestSchemaVersion is the version of the JSON shape emitted by
// `bb --describe`. Bumped independently of the CLI version when fields are
// added, removed, or change meaning. Agents should branch on this, not on
// the CLI's semver.
const manifestSchemaVersion = "1"

// Action classifies a command's idempotency for agent gating.
//
//	read        list, get, diff, activity, statuses, me
//	write       create, add, update, set, trigger, approve, merge, fork, reply, upload
//	destructive delete, decline, close, stop, complete (task), reopen
//
// Interactive commands (setup, auth login/logout) are not agent-callable and
// are marked `Skip: true` in commandRegistry rather than carrying an action.
const (
	actionRead        = "read"
	actionWrite       = "write"
	actionDestructive = "destructive"
)

// Stdin merge behaviors. Today every stdin-capable command uses
// stdinBehaviorReplacesFlags. The constant exists so the field is a stable
// enum on the wire (agents can switch on it) rather than free text.
const (
	stdinBehaviorReplacesFlags = "replaces_flags"
)

// describeFlag toggles manifest emission. Wired up in cmd/root.go init().
var describeFlag bool

// commandSpec is the registry entry for one leaf command. It carries the
// metadata that cannot be inferred from Cobra alone — action class, output
// type, optional stdin type, example invocation, and an ordering hint for
// list endpoints whose API does not accept a sort parameter.
//
// Registry entries live in cmd/manifest_registry.go.
type commandSpec struct {
	Action     string
	OutputType string // key into typeRegistry; "" allowed for write/destructive ops with no payload
	StdinType  string // key into typeRegistry; "" if command does not accept stdin
	Example    string
	Ordering   string // for list ops without --sort: stable_by_<field>, unspecified
	Skip       bool   // true for completion/setup — present in tree but not in manifest
}

// flagSpec is the wire shape for a single flag in the manifest.
type flagSpec struct {
	Name        string `json:"name"`
	Short       string `json:"short,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type stdinSpec struct {
	Type     string             `json:"type"`
	Schema   *jsonschema.Schema `json:"schema,omitempty"`
	Behavior string             `json:"behavior"`
}

type commandManifest struct {
	Path         []string           `json:"path"`
	Summary      string             `json:"summary"`
	Action       string             `json:"action"`
	Flags        []flagSpec         `json:"flags"`
	Stdin        *stdinSpec         `json:"stdin"`
	OutputType   string             `json:"output_type,omitempty"`
	OutputSchema *jsonschema.Schema `json:"output_schema,omitempty"`
	Ordering     string             `json:"ordering,omitempty"`
	Example      string             `json:"example,omitempty"`
}

type manifest struct {
	ManifestSchemaVersion string            `json:"manifest_schema_version"`
	Version               string            `json:"version"`
	Commands              []commandManifest `json:"commands"`
}

// buildManifest walks the Cobra tree under root, looks up each leaf in
// commandRegistry, and assembles the manifest. Returns an error if any leaf
// has no registry entry or references an unknown type — callers (and the
// invariant test) treat this as a build-time bug.
func buildManifest(root *cobra.Command, version string) (*manifest, error) {
	m := &manifest{
		ManifestSchemaVersion: manifestSchemaVersion,
		Version:               version,
	}
	var walkErr error
	visit(root, nil, func(path []string, c *cobra.Command) {
		if walkErr != nil {
			return
		}
		key := strings.Join(path, " ")
		spec, ok := commandRegistry[key]
		if !ok {
			walkErr = fmt.Errorf("describe: leaf command %q is missing from commandRegistry — add an entry in cmd/manifest_registry.go", key)
			return
		}
		if spec.Skip {
			return
		}
		entry, err := buildCommandManifest(path, c, spec)
		if err != nil {
			walkErr = err
			return
		}
		m.Commands = append(m.Commands, entry)
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Slice(m.Commands, func(i, j int) bool {
		return strings.Join(m.Commands[i].Path, " ") < strings.Join(m.Commands[j].Path, " ")
	})
	return m, nil
}

// visit calls fn on every leaf (no further subcommands) under root. The path
// argument accumulates Use tokens; the root's own Use is excluded.
func visit(c *cobra.Command, path []string, fn func([]string, *cobra.Command)) {
	subs := c.Commands()
	// Cobra adds `help` automatically — never include it in the manifest.
	leafSubs := make([]*cobra.Command, 0, len(subs))
	for _, s := range subs {
		if s.Name() == "help" {
			continue
		}
		leafSubs = append(leafSubs, s)
	}
	if len(leafSubs) == 0 {
		// A leaf: skip the synthetic root (no Use token of its own past "bb").
		if len(path) == 0 {
			return
		}
		fn(path, c)
		return
	}
	for _, s := range leafSubs {
		visit(s, append(append([]string{}, path...), s.Name()), fn)
	}
}

func buildCommandManifest(path []string, c *cobra.Command, spec commandSpec) (commandManifest, error) {
	entry := commandManifest{
		Path:    path,
		Summary: c.Short,
		Action:  spec.Action,
		Flags:   collectFlags(c),
		Example: spec.Example,
	}
	if spec.OutputType != "" {
		sample, ok := typeRegistry[spec.OutputType]
		if !ok {
			return entry, fmt.Errorf("describe: output type %q (for %s) not in typeRegistry", spec.OutputType, strings.Join(path, " "))
		}
		entry.OutputType = spec.OutputType
		entry.OutputSchema = reflectSchema(sample)
	}
	if spec.StdinType != "" {
		sample, ok := typeRegistry[spec.StdinType]
		if !ok {
			return entry, fmt.Errorf("describe: stdin type %q (for %s) not in typeRegistry", spec.StdinType, strings.Join(path, " "))
		}
		entry.Stdin = &stdinSpec{
			Type:     spec.StdinType,
			Schema:   reflectSchema(sample),
			Behavior: stdinBehaviorReplacesFlags,
		}
	}
	if spec.Ordering != "" {
		entry.Ordering = spec.Ordering
	}
	return entry, nil
}

// collectFlags walks both local and inherited flags. Persistent flags from the
// root (--config, --workspace, --token, --format, ...) are included so an agent
// has the full call surface in one place.
func collectFlags(c *cobra.Command) []flagSpec {
	out := []flagSpec{}
	seen := map[string]bool{}
	add := func(f *pflag.Flag) {
		// Hidden flags (eg cobra's --help) are excluded; --describe itself is
		// inherited but is only meaningful at the root, skip it on leaves.
		if f.Hidden || f.Name == "help" || f.Name == "describe" {
			return
		}
		if seen[f.Name] {
			return
		}
		seen[f.Name] = true
		_, required := f.Annotations[cobra.BashCompOneRequiredFlag]
		out = append(out, flagSpec{
			Name:        f.Name,
			Short:       f.Shorthand,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Required:    required,
			Description: f.Usage,
		})
	}
	c.LocalFlags().VisitAll(add)
	c.InheritedFlags().VisitAll(add)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// reflectSchema generates a JSON Schema for the type of sample. invopop's
// reflector by default emits $ref-laden schemas; we expand inline so the
// manifest is self-contained per-command.
func reflectSchema(sample any) *jsonschema.Schema {
	r := &jsonschema.Reflector{
		ExpandedStruct:             false,
		AllowAdditionalProperties:  true,
		DoNotReference:             true,
		RequiredFromJSONSchemaTags: false,
	}
	return r.ReflectFromType(reflect.TypeOf(sample))
}

// emitManifest writes the manifest as indented JSON to w and returns any
// marshalling error.
func emitManifest(w io.Writer, m *manifest) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

// runDescribe is invoked from rootCmd.PersistentPreRunE before any auth or
// config validation, so an agent can introspect without credentials. The root
// is passed in (rather than referencing the package-level rootCmd) to avoid
// an init cycle.
func runDescribe(root *cobra.Command) error {
	m, err := buildManifest(root, Version)
	if err != nil {
		return err
	}
	if err := emitManifest(os.Stdout, m); err != nil {
		return err
	}
	os.Exit(0)
	return nil // unreachable
}
