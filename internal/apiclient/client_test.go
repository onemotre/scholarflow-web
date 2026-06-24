package apiclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListPapersParses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/papers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`[{"paper_id":"p1","title":"T","status":"completed","uploaded_filename":"a.pdf"}]`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).ListPapers(context.Background())
	if err != nil {
		t.Fatalf("ListPapers: %v", err)
	}
	if len(got) != 1 || got[0].Status != "completed" || got[0].Title == nil || *got[0].Title != "T" {
		t.Fatalf("got = %#v", got)
	}
}

func TestGetPaperParsesCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"paper_id":"p1","status":"completed","uploaded_filename":"a.pdf",
		"card":{"background":"bg","method":"m","results":["r1"],
		"evidence":[{"claim_key":"method","evidence_type":"section","section_id":"3","snippet":"snip","confidence":0.8}]}}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).GetPaper(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetPaper: %v", err)
	}
	if got.Card == nil || got.Card.Method != "m" || len(got.Card.Evidence) != 1 || got.Card.Evidence[0].SectionID != "3" {
		t.Fatalf("card = %#v", got.Card)
	}
}

func TestGetPaperNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := New(Config{BaseURL: srv.URL}).GetPaper(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
