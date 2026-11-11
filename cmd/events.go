package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// Event lines are the --events output of the wait/watch commands: one
// logfmt-style `key=value ...` line per state change on stdout, then one final
// outcome line. They are meant for line-oriented consumers (an agent's Monitor
// tool, `while read`), which is why each event is exactly one line.

// eventField is one key=value pair of an event line.
type eventField struct {
	key   string
	value string
}

// formatEvent renders fields as a logfmt line. Empty values are skipped, and a
// value containing a space, quote, or '=' is quoted so the line stays parseable.
func formatEvent(fields ...eventField) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		parts = append(parts, f.key+"="+quoteEventValue(f.value))
	}
	return strings.Join(parts, " ")
}

func quoteEventValue(v string) string {
	if strings.ContainsAny(v, " \t\"=") {
		return strconv.Quote(v)
	}
	return v
}

// writeEvent writes one formatted event line to w.
func writeEvent(w io.Writer, fields ...eventField) {
	fmt.Fprintln(w, formatEvent(fields...))
}

// stepEvents emits a step line whenever a pipeline step's state changes
// between polls (and once for every step on the first poll).
type stepEvents struct {
	w    io.Writer
	last map[string]string // step key -> last emitted line
}

func newStepEvents(w io.Writer) *stepEvents {
	return &stepEvents{w: w, last: make(map[string]string)}
}

func (e *stepEvents) poll(steps []bitbucket.PipelineStep) {
	for _, s := range steps {
		line := stepEventLine(s)
		key := s.UUID
		if key == "" {
			key = s.Name
		}
		if e.last[key] == line {
			continue
		}
		e.last[key] = line
		fmt.Fprintln(e.w, line)
	}
}

func stepEventLine(s bitbucket.PipelineStep) string {
	fields := []eventField{{"step", s.Name}, {"state", s.State.Name}}
	if s.State.Stage != nil {
		fields = append(fields, eventField{"stage", s.State.Stage.Name})
	}
	if s.State.Result != nil {
		fields = append(fields, eventField{"result", s.State.Result.Name})
	}
	if s.Manual {
		fields = append(fields, eventField{"manual", "true"})
	}
	return formatEvent(fields...)
}

// pipelineOutcomeEvent is the final --events line of `pipeline watch`.
func pipelineOutcomeEvent(r bitbucket.PipelineWatchResult, url string) string {
	gate := ""
	if r.ManualGate != nil {
		gate = r.ManualGate.Step
	}
	return formatEvent(
		eventField{"pipeline", strconv.Itoa(r.Pipeline.BuildNumber)},
		eventField{"outcome", string(r.Status)},
		eventField{"gate", gate},
		eventField{"url", url},
	)
}

// waitInsteadOfLoops is the help-text guidance shared by every wait/watch
// command, so agents reach for them instead of hand-rolled polling loops.
const waitInsteadOfLoops = "To wait on a pipeline, PR build checks, or PR comments, use " +
	"`bb pipeline watch`, `bb pr wait-checks`, or `bb pr wait-comments`; do not write polling loops."
