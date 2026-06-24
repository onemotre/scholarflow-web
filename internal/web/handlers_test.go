package web

import (
	"context"
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
}

func (f *fakeAPI) ListPapers(ctx context.Context) ([]apiclient.PaperSummary, error) {
	return f.summaries, f.listErr
}
func (f *fakeAPI) GetPaper(ctx context.Context, id string) (apiclient.PaperDetail, error) {
	return f.detail, f.getErr
}

func routerFor(api API) http.Handler {
	r := chi.NewRouter()
	h := NewHandler(api)
	r.Get("/", h.Collection)
	r.Get("/papers/{id}", h.Paper)
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
		Card: &apiclient.Card{Method: "方法X", Evidence: []apiclient.Evidence{{ClaimKey: "method", SectionID: "3", Snippet: "片段"}}},
	}}).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/papers/p1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
	for _, want := range []string{"方法X", "sidenote", "片段"} {
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
