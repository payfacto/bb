package bitbucket_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestDownloads_List(t *testing.T) {
	downloads := []bitbucket.Download{
		{Name: "app-v1.0.0.zip", Size: 1048576},
		{Name: "app-v1.1.0.zip", Size: 2097152},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/downloads" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": downloads})
	}))
	got, err := client.Downloads("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "app-v1.0.0.zip" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[1].Size != 2097152 {
		t.Errorf("expected Size=2097152, got %d", got[1].Size)
	}
}

func TestDownloads_Upload(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/downloads" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		f, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("get form file 'files': %v", err)
		}
		defer f.Close()
		if header.Filename != "release.zip" {
			t.Errorf("expected filename=release.zip, got %s", header.Filename)
		}
		content, _ := io.ReadAll(f)
		if string(content) != "binary content" {
			t.Errorf("unexpected content: %s", content)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	err := client.Downloads("testws", "testrepo").Upload(context.Background(), "release.zip", strings.NewReader("binary content"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestDownloads_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/downloads/app-v1.0.0.zip" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Downloads("testws", "testrepo").Delete(context.Background(), "app-v1.0.0.zip")
	if err != nil {
		t.Fatal(err)
	}
}
