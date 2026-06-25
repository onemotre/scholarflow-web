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
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Caption  string `json:"caption"`
	Order    int32  `json:"order"`
	ID       string `json:"id"`
	HasImage bool   `json:"has_image"`
}

type Section struct {
	Order     int32   `json:"order"`
	Heading   *string `json:"heading,omitempty"`
	PageStart *int32  `json:"page_start,omitempty"`
	PageEnd   *int32  `json:"page_end,omitempty"`
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

type CardMethodology struct {
	Problem string `json:"problem"`
	Method  string `json:"method"`
}

type CardComparison struct {
	Work      string `json:"work"`
	Value     string `json:"value"`
	Reference string `json:"reference"`
}

type CardResult struct {
	Metric      string           `json:"metric"`
	Finding     string           `json:"finding"`
	Comparisons []CardComparison `json:"comparisons"`
	SelfOnly    bool             `json:"self_only"`
}

type CardModule struct {
	Name      string `json:"name"`
	Function  string `json:"function"`
	Design    string `json:"design"`
	Principle string `json:"principle"`
}

type CardImplementation struct {
	Overview string       `json:"overview"`
	Modules  []CardModule `json:"modules"`
}

type Card struct {
	Introduction   string             `json:"introduction"`
	RelatedWork    string             `json:"related_work"`
	Methodology    []CardMethodology  `json:"methodology"`
	Results        []CardResult       `json:"results"`
	Implementation CardImplementation `json:"implementation"`
	CodeLinks      []string           `json:"code_links"`
	DataLinks      []string           `json:"data_links"`
	Figures        []CardFigure       `json:"figures"`
	Evidence       []Evidence         `json:"evidence"`
}

type PaperDetail struct {
	PaperID          string    `json:"paper_id"`
	Status           string    `json:"status"`
	Title            *string   `json:"title,omitempty"`
	DOI              *string   `json:"doi,omitempty"`
	Abstract         *string   `json:"abstract,omitempty"`
	Sections         []Section `json:"sections"`
	PublicationYear  *int32    `json:"publication_year,omitempty"`
	UploadedFilename string    `json:"uploaded_filename"`
	Authors          []Author  `json:"authors"`
	Figures          []Figure  `json:"figures"`
	Card             *Card     `json:"card,omitempty"`
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

// GetFigureImage streams the API's figure-image endpoint. The caller must close
// the returned reader. Returns ErrNotFound on 404.
func (c *Client) GetFigureImage(ctx context.Context, paperID, figureID string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/papers/"+paperID+"/figures/"+figureID+"/image", nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, "", ErrNotFound
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("backend figure image returned %d", resp.StatusCode)
	}
	return resp.Body, resp.Header.Get("Content-Type"), nil
}
