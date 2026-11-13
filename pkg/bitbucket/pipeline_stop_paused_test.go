package bitbucket_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/payfacto/bb/pkg/bitbucket"
)

var (
	statePending = bitbucket.PipelineState{Name: "PENDING"}
	stateHalted  = bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "HALTED"}}
)

func build(n int, state bitbucket.PipelineState) bitbucket.Pipeline {
	return bitbucket.Pipeline{UUID: fmt.Sprintf("{p%d}", n), BuildNumber: n, State: state}
}

// pagedPipelines serves pages (newest first) with "next" links between them
// and counts the page requests.
func pagedPipelines(t *testing.T, pages [][]bitbucket.Pipeline, requests *int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests++
		index := 0
		if p := r.URL.Query().Get("page"); p != "" {
			index, _ = strconv.Atoi(p)
		}
		body := map[string]any{"values": pages[index]}
		if index+1 < len(pages) {
			body["next"] = fmt.Sprintf("http://%s%s?page=%d", r.Host, r.URL.Path, index+1)
		}
		mustEncodeJSON(t, w, body)
	})
}

func findSuperseded(t *testing.T, pages [][]bitbucket.Pipeline, maxPages int) (bitbucket.SupersededPaused, int) {
	t.Helper()
	requests := 0
	client := newTestClient(t, pagedPipelines(t, pages, &requests))
	got, err := client.Pipelines("ws", "repo").FindSupersededPaused(context.Background(), maxPages)
	if err != nil {
		t.Fatal(err)
	}
	return got, requests
}

func buildNumbers(pipelines []bitbucket.Pipeline) []int {
	out := []int{}
	for _, p := range pipelines {
		out = append(out, p.BuildNumber)
	}
	return out
}

func TestFindSupersededPaused_KeepsNewestActiveAndTargetsOlderGated(t *testing.T) {
	pages := [][]bitbucket.Pipeline{
		{build(9, stateSucceeded), build(8, statePaused), build(7, statePaused)},
		{build(6, stateRunning), build(5, stateHalted), build(4, stateStopped)},
	}
	got, _ := findSuperseded(t, pages, 10)
	if got.Kept == nil || got.Kept.BuildNumber != 8 {
		t.Fatalf("Kept = %+v, want build 8 (newest not-completed)", got.Kept)
	}
	if want, have := []int{7, 5}, buildNumbers(got.Targets); fmt.Sprint(have) != fmt.Sprint(want) {
		t.Errorf("Targets = %v, want %v (older running build 6 and completed builds untouched)", have, want)
	}
	if got.Scanned != 6 {
		t.Errorf("Scanned = %d, want 6", got.Scanned)
	}
	if got.Truncated {
		t.Error("Truncated = true, want false: every page was read")
	}
}

func TestFindSupersededPaused_QueuedBuildIsTheKeeper(t *testing.T) {
	pages := [][]bitbucket.Pipeline{{build(3, statePending), build(2, statePaused), build(1, statePaused)}}
	got, _ := findSuperseded(t, pages, 10)
	if got.Kept == nil || got.Kept.BuildNumber != 3 {
		t.Fatalf("Kept = %+v, want queued build 3", got.Kept)
	}
	if got.KeptPaused == nil || got.KeptPaused.BuildNumber != 2 {
		t.Fatalf("KeptPaused = %+v, want build 2: the queued build has not reached its gate", got.KeptPaused)
	}
	if want, have := []int{1}, buildNumbers(got.Targets); fmt.Sprint(have) != fmt.Sprint(want) {
		t.Errorf("Targets = %v, want %v", have, want)
	}
}

func TestFindSupersededPaused_NoActiveBuildMeansNothingToStop(t *testing.T) {
	pages := [][]bitbucket.Pipeline{{build(2, stateSucceeded), build(1, stateStopped)}}
	got, _ := findSuperseded(t, pages, 10)
	if got.Kept != nil {
		t.Errorf("Kept = %+v, want nil", got.Kept)
	}
	if len(got.Targets) != 0 {
		t.Errorf("Targets = %v, want none", buildNumbers(got.Targets))
	}
}

func TestFindSupersededPaused_StopsAtMaxPagesAndReportsTruncation(t *testing.T) {
	pages := [][]bitbucket.Pipeline{
		{build(6, statePaused)},
		{build(5, statePaused)},
		{build(4, statePaused)},
	}
	got, requests := findSuperseded(t, pages, 2)
	if requests != 2 {
		t.Errorf("fetched %d pages, want 2", requests)
	}
	if !got.Truncated {
		t.Error("Truncated = false, want true: a third page exists")
	}
	if want, have := []int{5}, buildNumbers(got.Targets); fmt.Sprint(have) != fmt.Sprint(want) {
		t.Errorf("Targets = %v, want %v", have, want)
	}
}

// fleet fakes several pipelines for StopPaused, keyed by UUID. Each pipeline
// reads as before until it receives a stop, then as after. A UUID listed in
// failStop answers the stop request with 500.
type fleet struct {
	before, after map[string]bitbucket.PipelineState
	failStop      map[string]bool
	stopped       map[string]bool
}

func (f *fleet) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := ""
		for id := range f.before {
			if strings.Contains(r.URL.Path, id) {
				uuid = id
			}
		}
		if uuid == "" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			return
		}
		if r.Method == http.MethodPost {
			if f.failStop[uuid] {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			f.stopped[uuid] = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		state := f.before[uuid]
		if f.stopped[uuid] {
			state = f.after[uuid]
		}
		n, _ := strconv.Atoi(strings.Trim(uuid, "{p}"))
		mustEncodeJSON(t, w, build(n, state))
	})
}

func TestStopPaused_SortsEachTargetIntoStoppedSkippedOrFailed(t *testing.T) {
	f := &fleet{
		before: map[string]bitbucket.PipelineState{
			"{p7}": statePaused,    // stops cleanly
			"{p6}": stateRunning,   // resumed since the scan: skipped, never stopped
			"{p5}": stateSucceeded, // finished since the scan: skipped
			"{p4}": statePaused,    // stop request fails
			"{p3}": statePaused,    // stays paused after the stop
		},
		after: map[string]bitbucket.PipelineState{
			"{p7}": stateStopped, "{p3}": statePaused,
		},
		failStop: map[string]bool{"{p4}": true},
		stopped:  map[string]bool{},
	}
	client := newTestClient(t, f.handler(t))
	targets := []bitbucket.Pipeline{
		build(7, statePaused), build(6, statePaused), build(5, statePaused),
		build(4, statePaused), build(3, statePaused),
	}
	opts := bitbucket.StopOptions{Interval: time.Millisecond, Timeout: 20 * time.Millisecond}
	got := client.Pipelines("ws", "repo").StopPaused(context.Background(), targets, opts)

	if len(got.Stopped) != 1 || got.Stopped[0].BuildNumber != 7 || got.Stopped[0].Result != bitbucket.StopConfirmed {
		t.Errorf("Stopped = %+v, want build 7 stopped", got.Stopped)
	}
	if fmt.Sprint(issueNumbers(got.Skipped)) != "[6 5]" {
		t.Errorf("Skipped = %+v, want builds 6 and 5", got.Skipped)
	}
	if fmt.Sprint(issueNumbers(got.Failed)) != "[4 3]" {
		t.Errorf("Failed = %+v, want builds 4 and 3", got.Failed)
	}
	if f.stopped["{p6}"] || f.stopped["{p5}"] {
		t.Error("sent a stop to a pipeline that was no longer paused")
	}
	if !got.HasFailures() {
		t.Error("HasFailures() = false, want true")
	}
}

func issueNumbers(items []bitbucket.StopIssue) []int {
	out := []int{}
	for _, i := range items {
		out = append(out, i.BuildNumber)
	}
	return out
}

func TestNewStopPausedReport_DryRunListsAreEmptyNotNull(t *testing.T) {
	report := bitbucket.NewStopPausedReport(bitbucket.SupersededPaused{})
	report.DryRun = true
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"targets":[]`, `"stopped":[]`, `"skipped":[]`, `"failed":[]`, `"dry_run":true`} {
		if !strings.Contains(string(data), key) {
			t.Errorf("report JSON %s lacks %s", data, key)
		}
	}
}

func TestStopPaused_CancelledContextFailsTheRemainingTargetsWithoutRequests(t *testing.T) {
	requests := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	targets := []bitbucket.Pipeline{build(2, statePaused), build(1, statePaused)}
	got := client.Pipelines("ws", "repo").StopPaused(ctx, targets, bitbucket.StopOptions{})
	if fmt.Sprint(issueNumbers(got.Failed)) != "[2 1]" {
		t.Errorf("Failed = %+v, want builds 2 and 1", got.Failed)
	}
	if requests != 0 {
		t.Errorf("sent %d requests after cancellation, want 0", requests)
	}
}

func TestFindSupersededPaused_NewestPausedBuildIsKeptWhenANewerBuildIsRunning(t *testing.T) {
	pages := [][]bitbucket.Pipeline{{build(5, stateRunning), build(4, statePaused), build(3, stateHalted)}}
	got, _ := findSuperseded(t, pages, 10)
	if got.Kept == nil || got.Kept.BuildNumber != 5 {
		t.Fatalf("Kept = %+v, want running build 5", got.Kept)
	}
	if got.KeptPaused == nil || got.KeptPaused.BuildNumber != 4 {
		t.Fatalf("KeptPaused = %+v, want build 4 (the one ready to deploy)", got.KeptPaused)
	}
	if want, have := []int{3}, buildNumbers(got.Targets); fmt.Sprint(have) != fmt.Sprint(want) {
		t.Errorf("Targets = %v, want %v", have, want)
	}
}

func TestFindSupersededPaused_PausedKeeperIsNotReportedTwice(t *testing.T) {
	pages := [][]bitbucket.Pipeline{{build(4, statePaused), build(3, statePaused)}}
	got, _ := findSuperseded(t, pages, 10)
	if got.KeptPaused != nil {
		t.Errorf("KeptPaused = %+v, want nil: the keeper is already the newest paused build", got.KeptPaused)
	}
	if want, have := []int{3}, buildNumbers(got.Targets); fmt.Sprint(have) != fmt.Sprint(want) {
		t.Errorf("Targets = %v, want %v", have, want)
	}
}
