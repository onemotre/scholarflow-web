package web

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"scholarflow_web/internal/apiclient"
)

type API interface {
	ListPapers(ctx context.Context) ([]apiclient.PaperSummary, error)
	GetPaper(ctx context.Context, id string) (apiclient.PaperDetail, error)
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
	view := PaperView{Detail: detail}
	if detail.Card != nil {
		view.EvidenceByClaim = GroupEvidenceByClaim(detail.Card.Evidence)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := Render(w, "paper.tmpl", view); err != nil {
		log.Printf("render paper: %v", err)
	}
}

func (h *Handler) renderError(w http.ResponseWriter, status int, heading, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = Render(w, "error.tmpl", map[string]string{"Heading": heading, "Message": message})
}
