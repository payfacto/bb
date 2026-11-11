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

// changeEvents emits a line for an item (a pipeline step, a build check)
// whenever its rendered line differs from the last one emitted for it, so every
// item is reported once on the first poll and then only on change.
type changeEvents struct {
	w    io.Writer
	last map[string]string // item key -> last emitted line
}

func newChangeEvents(w io.Writer) *changeEvents {
	return &changeEvents{w: w, last: make(map[string]string)}
}

func (e *changeEvents) emit(key, line string) {
	if e.last[key] == line {
		return
	}
	e.last[key] = line
	fmt.Fprintln(e.w, line)
}

// steps reports pipeline step state changes.
func (e *changeEvents) steps(steps []bitbucket.PipelineStep) {
	for _, s := range steps {
		e.emit(firstNonEmpty(s.UUID, s.Name), stepEventLine(s))
	}
}

// checks reports PR build status changes.
func (e *changeEvents) checks(statuses []bitbucket.PRStatus) {
	for _, s := range statuses {
		e.emit(firstNonEmpty(s.Key, s.Name), formatEvent(
			eventField{"check", firstNonEmpty(s.Name, s.Key)},
			eventField{"state", s.State},
		))
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
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

// pipelineOutcomeEvent is the final --events line of `pipeline watch`. build is
// the watched pipeline's number, which the result lacks when the watch timed
// out before its first successful poll.
func pipelineOutcomeEvent(r bitbucket.PipelineWatchResult, build int, url string) string {
	gate := ""
	if r.ManualGate != nil {
		gate = r.ManualGate.Step
	}
	return formatEvent(
		eventField{"pipeline", strconv.Itoa(build)},
		eventField{"outcome", string(r.Status)},
		eventField{"gate", gate},
		eventField{"url", url},
	)
}

// waitInsteadOfLoops is the help-text guidance shared by every wait/watch
// command, so agents reach for them instead of hand-rolled polling loops.
const waitInsteadOfLoops = "To wait on a pipeline, PR build checks, or PR comments, use " +
	"`bb pipeline watch`, `bb pr wait-checks`, or `bb pr wait-comments`; do not write polling loops."
