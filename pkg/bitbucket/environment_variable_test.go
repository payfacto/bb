package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestEnvironmentVariables_List(t *testing.T) {
	vars := []bitbucket.PipelineVariable{
		{UUID: "{v-1}", Key: "API_URL", Value: "https://x", Secured: false},
		{UUID: "{v-2}", Key: "SECRET", Value: "", Secured: true},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		mustEncodeJSON(t, w, map[string]any{"values": vars})
	}))
	got, err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Key != "SECRET" || !got[1].Secured {
		t.Errorf("unexpected result: %+v", got)
	}
	_ = json.Marshal // ensure encoding/json is used (referenced by later tasks)
}
