package apiclient

import (
	"context"
	"errors"
	"io"
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
		"abstract":"原始摘要",
		"sections":[{"order":1,"heading":"Introduction","page_start":1,"page_end":2}],
		"figures":[{"label":"Figure 2","kind":"figure","caption":"结构图","order":2,"id":"fig-2","has_image":true}],
		"card":{"introduction":"引言","related_work":"相关","methodology":[{"problem":"P","method":"M"}],
		"results":[{"metric":"acc","finding":"更好","comparisons":[{"work":"BaseX","value":"80%","reference":"[12]"}],"self_only":false}],
		"implementation":{"overview":"总体","modules":[{"name":"Enc","function":"编码","design":"D","principle":"E=mc^2"}]},
		"evidence":[{"claim_key":"results","claim_index":0,"evidence_type":"section","section_id":"3","page":7,"snippet":"snip","confidence":0.8}]}}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).GetPaper(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetPaper: %v", err)
	}
	if got.Abstract == nil || *got.Abstract != "原始摘要" {
		t.Fatalf("abstract = %#v", got.Abstract)
	}
	if len(got.Sections) != 1 || got.Sections[0].Heading == nil || *got.Sections[0].Heading != "Introduction" {
		t.Fatalf("sections = %#v", got.Sections)
	}
	if len(got.Figures) != 1 || got.Figures[0].ID != "fig-2" || !got.Figures[0].HasImage {
		t.Fatalf("figures = %#v", got.Figures)
	}
	c := got.Card
	if c == nil || c.Introduction != "引言" || len(c.Methodology) != 1 || c.Methodology[0].Method != "M" {
		t.Fatalf("card = %#v", c)
	}
	if len(c.Results) != 1 || c.Results[0].Metric != "acc" || len(c.Results[0].Comparisons) != 1 || c.Results[0].Comparisons[0].Work != "BaseX" {
		t.Fatalf("results = %#v", c.Results)
	}
	if c.Implementation.Overview != "总体" || len(c.Implementation.Modules) != 1 || c.Implementation.Modules[0].Name != "Enc" {
		t.Fatalf("implementation = %#v", c.Implementation)
	}
	if len(c.Evidence) != 1 || c.Evidence[0].SectionID != "3" {
		t.Fatalf("evidence = %#v", c.Evidence)
	}
}

func TestGetFigureImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/papers/p1/figures/f2/image" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("PNGBYTES"))
	}))
	defer srv.Close()

	body, ct, err := New(Config{BaseURL: srv.URL}).GetFigureImage(context.Background(), "p1", "f2")
	if err != nil {
		t.Fatalf("GetFigureImage: %v", err)
	}
	defer body.Close()
	if ct != "image/png" {
		t.Fatalf("content-type = %q", ct)
	}
	data, _ := io.ReadAll(body)
	if string(data) != "PNGBYTES" {
		t.Fatalf("body = %q", string(data))
	}
}

func TestGetFigureImageNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	_, _, err := New(Config{BaseURL: srv.URL}).GetFigureImage(context.Background(), "p1", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
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
