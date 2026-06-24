package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNotFound is returned when the backend responds 404.
var ErrNotFound = errors.New("not found")

type Config struct {
	BaseURL string
	Timeout time.Duration
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

type PaperSummary struct {
	PaperID          string     `json:"paper_id"`
	Title            *string    `json:"title,omitempty"`
	Status           string     `json:"status"`
	PublicationYear  *int32     `json:"publication_year,omitempty"`
	UploadedFilename string     `json:"uploaded_filename"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
}

type Author struct {
	Order       int32   `json:"order"`
	DisplayName string  `json:"display_name"`
	ORCID       *string `json:"orcid,omitempty"`
}

type Figure struct {
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Caption string `json:"caption"`
	Order   int32  `json:"order"`
}

type Evidence struct {
	ClaimKey     string  `json:"claim_key"`
	ClaimIndex   *int    `json:"claim_index,omitempty"`
	EvidenceType string  `json:"evidence_type"`
	SectionID    string  `json:"section_id,omitempty"`
	Snippet      string  `json:"snippet,omitempty"`
	Page         *int    `json:"page,omitempty"`
	Confidence   float64 `json:"confidence"`
}

// CardFigure places a figure (by label) at a claim anchor. Caption is not part
// of the card; the viewer resolves it from PaperDetail.Figures by label.
type CardFigure struct {
	Label      string `json:"label"`
	ClaimKey   string `json:"claim_key"`
	ClaimIndex *int   `json:"claim_index,omitempty"`
	Page       *int   `json:"page,omitempty"`
}

type Card struct {
	Background     string       `json:"background"`
	Problem        string       `json:"problem"`
	Method         string       `json:"method"`
	Implementation string       `json:"implementation"`
	Benchmarks     []string     `json:"benchmarks"`
	Baselines      []string     `json:"baselines"`
	Results        []string     `json:"results"`
	CodeLinks      []string     `json:"code_links"`
	DataLinks      []string     `json:"data_links"`
	Figures        []CardFigure `json:"figures"`
	Evidence       []Evidence   `json:"evidence"`
}

type PaperDetail struct {
	PaperID          string   `json:"paper_id"`
	Status           string   `json:"status"`
	Title            *string  `json:"title,omitempty"`
	DOI              *string  `json:"doi,omitempty"`
	PublicationYear  *int32   `json:"publication_year,omitempty"`
	UploadedFilename string   `json:"uploaded_filename"`
	Authors          []Author `json:"authors"`
	Figures          []Figure `json:"figures"`
	Card             *Card    `json:"card,omitempty"`
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) ListPapers(ctx context.Context) ([]PaperSummary, error) {
	var summaries []PaperSummary
	if err := c.get(ctx, "/v1/papers", &summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

func (c *Client) GetPaper(ctx context.Context, id string) (PaperDetail, error) {
	var detail PaperDetail
	if err := c.get(ctx, "/v1/papers/"+id, &detail); err != nil {
		return PaperDetail{}, err
	}
	return detail, nil
}
