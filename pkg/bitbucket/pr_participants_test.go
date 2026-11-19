package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const prWithParticipantsJSON = `{
  "id": 2, "title": "t", "state": "OPEN",
  "reviewers": [],
  "participants": [
    {"user": {"account_id": "acc-1", "uuid": "{u-1}", "display_name": "Petra"},
     "role": "PARTICIPANT", "approved": false, "state": null,
     "participated_on": "2026-11-19T18:11:53+00:00"},
    {"user": {"account_id": "acc-2", "uuid": "{u-2}", "display_name": "Rev"},
     "role": "REVIEWER", "approved": true, "state": "approved",
     "participated_on": null}
  ]
}`

func TestPRs_Get_DecodesParticipants(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(prWithParticipantsJSON))
	}))
	pr, err := c.PRs("ws", "repo").Get(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pr.Participants) != 2 {
		t.Fatalf("participants: got %d, want 2", len(pr.Participants))
	}
	first, second := pr.Participants[0], pr.Participants[1]
	if first.User.AccountID != "acc-1" || first.User.UUID != "{u-1}" || first.Role != "PARTICIPANT" {
		t.Errorf("first participant: %+v", first)
	}
	if first.State != nil || first.Approved {
		t.Errorf("first participant should be unapproved with nil state: %+v", first)
	}
	if first.ParticipatedOn != "2026-11-19T18:11:53+00:00" {
		t.Errorf("participated_on: got %q", first.ParticipatedOn)
	}
	if second.State == nil || *second.State != "approved" || !second.Approved || second.Role != "REVIEWER" {
		t.Errorf("second participant: %+v", second)
	}
}

func TestPRs_List_RequestsReviewersAndParticipants(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("fields"), "+values.reviewers,+values.participants"; got != want {
			t.Errorf("fields: got %q, want %q", got, want)
		}
		_, _ = w.Write([]byte(`{"values": [` + prWithParticipantsJSON + `]}`))
	}))
	got, err := c.PRs("ws", "repo").List(context.Background(), bitbucket.PRListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || len(got[0].Participants) != 2 {
		t.Errorf("unexpected PRs: %+v", got)
	}
}

func TestPRs_List_ReviewerFilter(t *testing.T) {
	tests := []struct {
		name, reviewer, wantQ string
	}{
		{"account id", "acc-9", `state="OPEN" AND reviewers.account_id="acc-9"`},
		{"uuid", "{u-9}", `state="OPEN" AND reviewers.uuid="{u-9}"`},
		{"quote is escaped", `a"b`, `state="OPEN" AND reviewers.account_id="a\"b"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("q"); got != tc.wantQ {
					t.Errorf("q: got %q, want %q", got, tc.wantQ)
				}
				mustEncodeJSON(t, w, map[string]any{"values": []bitbucket.PR{}})
			}))
			_, err := c.PRs("ws", "repo").List(context.Background(),
				bitbucket.PRListOptions{State: "OPEN", Reviewer: tc.reviewer})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func participantPR(id int, accountID, uuid string) bitbucket.PR {
	return bitbucket.PR{ID: id, Participants: []bitbucket.Participant{
		{User: bitbucket.Actor{AccountID: accountID, UUID: uuid}, Role: "PARTICIPANT"},
	}}
}

func TestPRs_List_ParticipantFilterIsClientSide(t *testing.T) {
	values := []bitbucket.PR{
		participantPR(1, "acc-me", "{u-me}"),
		participantPR(2, "acc-other", "{u-other}"),
		participantPR(3, "acc-me", "{u-me}"),
		{ID: 4},
	}
	tests := []struct {
		name, participant string
		limit             int
		wantIDs           []int
	}{
		{"by account id", "acc-me", 0, []int{1, 3}},
		{"by uuid", "{u-me}", 0, []int{1, 3}},
		{"limit counts matches, not fetched PRs", "acc-me", 1, []int{1}},
		{"no match", "acc-nobody", 0, []int{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if q := r.URL.Query().Get("q"); q != "" {
					t.Errorf("participant must not be sent in q (API rejects it), got %q", q)
				}
				mustEncodeJSON(t, w, map[string]any{"values": values})
			}))
			got, err := c.PRs("ws", "repo").List(context.Background(),
				bitbucket.PRListOptions{Participant: tc.participant, Limit: tc.limit})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("result must be a non-nil slice so JSON prints []")
			}
			ids := []int{}
			for _, pr := range got {
				ids = append(ids, pr.ID)
			}
			if len(ids) != len(tc.wantIDs) {
				t.Fatalf("ids: got %v, want %v", ids, tc.wantIDs)
			}
			for i := range ids {
				if ids[i] != tc.wantIDs[i] {
					t.Fatalf("ids: got %v, want %v", ids, tc.wantIDs)
				}
			}
		})
	}
}

func TestUsers_Me_DecodesUUID(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"account_id": "acc-me", "uuid": "{u-me}", "nickname": "me"}`))
	}))
	me, err := c.User().Me(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if me.UUID != "{u-me}" {
		t.Errorf("uuid: got %q, want {u-me}", me.UUID)
	}
}
