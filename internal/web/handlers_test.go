package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"scholarflow_web/internal/apiclient"
)

type fakeAPI struct {
	summaries []apiclient.PaperSummary
	detail    apiclient.PaperDetail
	listErr   error
	getErr    error
	imgBody   string
	imgType   string
	imgErr    error
}

func (f *fakeAPI) ListPapers(ctx context.Context) ([]apiclient.PaperSummary, error) {
	return f.summaries, f.listErr
}
func (f *fakeAPI) GetPaper(ctx context.Context, id string) (apiclient.PaperDetail, error) {
	return f.detail, f.getErr
}
func (f *fakeAPI) GetFigureImage(ctx context.Context, paperID, figureID string) (io.ReadCloser, string, error) {
	if f.imgErr != nil {
		return nil, "", f.imgErr
	}
	return io.NopCloser(strings.NewReader(f.imgBody)), f.imgType, nil
}

func routerFor(api API) http.Handler {
	r := chi.NewRouter()
	h := NewHandler(api)
	r.Get("/", h.Collection)
	r.Get("/papers/{id}", h.Paper)
	r.Get("/papers/{id}/figures/{figureId}/image", h.FigureImage)
	return r
}

func TestCollectionRendersRows(t *testing.T) {
	title := "标题A"
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{summaries: []apiclient.PaperSummary{{PaperID: "p1", Title: &title, Status: "completed", UploadedFilename: "a.pdf"}}}).
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "/papers/p1") {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPaperRendersCard(t *testing.T) {
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{detail: apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Card: &apiclient.Card{Introduction: "引言X", Evidence: []apiclient.Evidence{{ClaimKey: "introduction", SectionID: "3", Snippet: "片段"}}},
	}}).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/papers/p1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
	for _, want := range []string{"引言X", "claim-notes", "片段"} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestPaperNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{getErr: apiclient.ErrNotFound}).
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/papers/missing", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404", rr.Code)
	}
}

func TestPaperBackendError(t *testing.T) {
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{getErr: context.DeadlineExceeded}).
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/papers/p1", nil))
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("code=%d, want 502", rr.Code)
	}
}

func TestCollectionBackendError(t *testing.T) {
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{listErr: context.DeadlineExceeded}).
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("code=%d, want 502", rr.Code)
	}
}

func TestFigureImageProxies(t *testing.T) {
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{imgBody: "PNGBYTES", imgType: "image/png"}).
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/papers/p1/figures/f2/image", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type = %q", ct)
	}
	if rr.Body.String() != "PNGBYTES" {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestFigureImageNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{imgErr: apiclient.ErrNotFound}).
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/papers/p1/figures/missing/image", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rr.Code)
	}
}
