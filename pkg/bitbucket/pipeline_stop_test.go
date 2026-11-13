package bitbucket_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const stopTestUUID = "{abc-123}"

var (
	stateRunning   = bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "RUNNING"}}
	statePaused    = bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "PAUSED"}}
	stateStopped   = bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "STOPPED"}}
	stateSucceeded = bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}}
)

// stopServer fakes Bitbucket for StopAndConfirm. GETs return before until the
// stop POST arrives, then after. It counts POSTs and post-stop GETs.
type stopServer struct {
	mu        sync.Mutex
	before    bitbucket.PipelineState
	after     bitbucket.PipelineState
	posts     int
	getsAfter int
	// failGetsAfter makes the first N post-stop GETs answer 404, like the
	// intermittent 404s Bitbucket was seen returning for pipeline reads.
	failGetsAfter int
}

func (s *stopServer) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/stopPipeline"):
			s.posts++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet:
			state := s.before
			if s.posts > 0 {
				state = s.after
				s.getsAfter++
				if s.getsAfter <= s.failGetsAfter {
					w.WriteHeader(http.StatusNotFound)
					return
				}
			}
			mustEncodeJSON(t, w, bitbucket.Pipeline{UUID: stopTestUUID, BuildNumber: 7, State: state})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
}

func runStop(t *testing.T, srv *stopServer, timeout time.Duration) (bitbucket.StopResult, error) {
	t.Helper()
	client := newTestClient(t, srv.handler(t))
	opts := bitbucket.StopOptions{Interval: time.Millisecond, Timeout: timeout}
	return client.Pipelines("ws", "repo").StopAndConfirm(context.Background(), stopTestUUID, opts)
}

func TestStopAndConfirm_RunningPipelineReachesStopped(t *testing.T) {
	srv := &stopServer{before: stateRunning, after: stateStopped}
	got, err := runStop(t, srv, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result != bitbucket.StopConfirmed {
		t.Errorf("Result = %q, want %q", got.Result, bitbucket.StopConfirmed)
	}
	if got.State.Result == nil || got.State.Result.Name != "STOPPED" {
		t.Errorf("State = %+v, want COMPLETED/STOPPED", got.State)
	}
	if got.BuildNumber != 7 {
		t.Errorf("BuildNumber = %d, want 7", got.BuildNumber)
	}
	if got.UUID != stopTestUUID {
		t.Errorf("UUID = %q, want %q", got.UUID, stopTestUUID)
	}
}

func TestStopAndConfirm_PausedPipelineThatStaysPausedIsReported(t *testing.T) {
	srv := &stopServer{before: statePaused, after: statePaused}
	got, err := runStop(t, srv, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result != bitbucket.StopStillPaused {
		t.Errorf("Result = %q, want %q", got.Result, bitbucket.StopStillPaused)
	}
}

func TestStopAndConfirm_StillRunningAtTimeoutIsOnlyRequested(t *testing.T) {
	srv := &stopServer{before: stateRunning, after: stateRunning}
	got, err := runStop(t, srv, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result != bitbucket.StopRequested {
		t.Errorf("Result = %q, want %q", got.Result, bitbucket.StopRequested)
	}
}

func TestStopAndConfirm_ZeroTimeoutSkipsConfirmation(t *testing.T) {
	srv := &stopServer{before: stateRunning, after: stateStopped}
	got, err := runStop(t, srv, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result != bitbucket.StopRequested {
		t.Errorf("Result = %q, want %q", got.Result, bitbucket.StopRequested)
	}
	if srv.getsAfter != 0 {
		t.Errorf("polled %d times after stop, want 0", srv.getsAfter)
	}
}

func TestStopAndConfirm_AlreadyCompletedIsRefusedWithoutStopping(t *testing.T) {
	srv := &stopServer{before: stateSucceeded, after: stateSucceeded}
	_, err := runStop(t, srv, time.Second)
	if !errors.Is(err, bitbucket.ErrPipelineAlreadyCompleted) {
		t.Fatalf("err = %v, want ErrPipelineAlreadyCompleted", err)
	}
	if srv.posts != 0 {
		t.Errorf("sent %d stop requests, want 0", srv.posts)
	}
}

func TestStopAndConfirm_TransientReadErrorAfterStopKeepsPolling(t *testing.T) {
	srv := &stopServer{before: stateRunning, after: stateStopped, failGetsAfter: 2}
	got, err := runStop(t, srv, time.Second)
	if err != nil {
		t.Fatalf("err = %v, want nil: the stop was already sent", err)
	}
	if got.Result != bitbucket.StopConfirmed {
		t.Errorf("Result = %q, want %q", got.Result, bitbucket.StopConfirmed)
	}
}

func TestStopAndConfirm_ReadsFailingUntilTimeoutReportStopRequested(t *testing.T) {
	srv := &stopServer{before: stateRunning, after: stateStopped, failGetsAfter: 1 << 30}
	got, err := runStop(t, srv, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("err = %v, want nil: the stop was already sent", err)
	}
	if got.Result != bitbucket.StopRequested {
		t.Errorf("Result = %q, want %q", got.Result, bitbucket.StopRequested)
	}
}

func TestStopAndConfirm_PausedWithNoReadAfterStopIsNotReportedStillPaused(t *testing.T) {
	srv := &stopServer{before: statePaused, after: stateStopped, failGetsAfter: 1 << 30}
	got, err := runStop(t, srv, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result != bitbucket.StopRequested {
		t.Errorf("Result = %q, want %q: no read after the stop, so paused is unproven", got.Result, bitbucket.StopRequested)
	}
}
