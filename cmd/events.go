package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

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

// formatEvent renders fields as a single logfmt line. Empty values are skipped
// and ambiguous values are quoted (see quoteEventValue).
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

// quoteEventValue Go-quotes a value that would otherwise be ambiguous: one
// with a space, quote, '=' or backslash, or any control or non-printable rune.
// Quoting escapes \n and \r, so API text can never split one event into two.
func quoteEventValue(v string) string {
	if strings.IndexFunc(v, needsEventQuoting) >= 0 {
		return strconv.Quote(v)
	}
	return v
}

func needsEventQuoting(r rune) bool {
	return r == ' ' || r == '"' || r == '=' || r == '\\' || !unicode.IsPrint(r)
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

// prOutcomeEvent is the final --events line of the PR wait commands.
func prOutcomeEvent(prID int, outcome string, extra ...eventField) string {
	fields := append([]eventField{{"pr", strconv.Itoa(prID)}, {"outcome", outcome}}, extra...)
	return formatEvent(fields...)
}
