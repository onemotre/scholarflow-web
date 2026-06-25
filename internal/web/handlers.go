package web

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"scholarflow_web/internal/apiclient"
)

type API interface {
	ListPapers(ctx context.Context) ([]apiclient.PaperSummary, error)
	GetPaper(ctx context.Context, id string) (apiclient.PaperDetail, error)
	GetFigureImage(ctx context.Context, paperID, figureID string) (io.ReadCloser, string, error)
}

type Handler struct {
	api API
}

func NewHandler(api API) *Handler {
	return &Handler{api: api}
}

func (h *Handler) Collection(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.api.ListPapers(r.Context())
	if err != nil {
		h.renderError(w, http.StatusBadGateway, "后端不可用", "无法从 API 获取论文列表。")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := Render(w, "collection.tmpl", summaries); err != nil {
		log.Printf("render collection: %v", err)
	}
}

func (h *Handler) Paper(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.api.GetPaper(r.Context(), id)
	if err != nil {
		if errors.Is(err, apiclient.ErrNotFound) {
			h.renderError(w, http.StatusNotFound, "未找到", "没有这篇论文。")
			return
		}
		h.renderError(w, http.StatusBadGateway, "后端不可用", "无法从 API 获取论文详情。")
		return
	}
	view := BuildPaperView(detail)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := Render(w, "paper.tmpl", view); err != nil {
		log.Printf("render paper: %v", err)
	}
}

func (h *Handler) FigureImage(w http.ResponseWriter, r *http.Request) {
	paperID := chi.URLParam(r, "id")
	figureID := chi.URLParam(r, "figureId")
	body, contentType, err := h.api.GetFigureImage(r.Context(), paperID, figureID)
	if err != nil {
		if errors.Is(err, apiclient.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.renderError(w, http.StatusBadGateway, "后端不可用", "无法获取图片。")
		return
	}
	defer body.Close()
	if contentType == "" {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = io.Copy(w, body)
}

func (h *Handler) renderError(w http.ResponseWriter, status int, heading, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = Render(w, "error.tmpl", map[string]string{"Heading": heading, "Message": message})
}
