# Web Viewer v3 — Design

**Date:** 2026-06-24
**Status:** Approved (brainstorming)
**Module:** `scholarflow-web`

## Context

Part of the larger paper-reading quality effort. The server now produces v3 paper
cards (schema 3.0) and extracts figure images (P1), but the read-only HTML viewer
(`scholarflow-web`, Tufte CSS, evidence as sidenotes) still renders the old 2.0 card
fields. This slice updates the viewer to render v3 and adds four reader-requested
features.

The viewer is a server-side-rendered Go-template app (chi router, `html/template`,
embedded templates, no client-side JS — interactivity is CSS-only, e.g. the Tufte
checkbox toggle). It reaches the API over a host-published port
(`SCHOLARFLOW_API_URL`) and never touches the database.

### Key finding: no server change needed

`GET /v1/papers/{id}` already returns everything required:

- `abstract` (`*string`) — for demand 2.
- `sections` (`[{order, heading, page_start, page_end}]`) — for the outline (demand 1).
- `figures` with `id` and `has_image` (added in P1) — for demand 3.
- The figure-image endpoint `GET /v1/papers/{id}/figures/{figureId}/image` exists.

The web's `apiclient` simply does not deserialize these yet. So this is a
**web-only** slice.

## Demands (from the user)

1. **Thesis outline** — render the paper's actual section structure (from the parsed
   `sections`: headings + page numbers), placed at the top of the reading page.
2. **Original abstract** — render the paper's verbatim abstract near the top.
3. **Figure images** — render the P1-extracted images: the architecture diagram
   (the figure anchored at `claim_key:"implementation"`) **inline** in the reading
   column; all other figures (charts at result anchors) as **margin thumbnails with
   click-to-enlarge**.
4. **Comment truncation + hover** — each evidence sidenote is truncated to one line
   by default, but its sequence number, section (`[§N]`), and page (`[p.N]`) are
   always shown; hovering the comment expands it in place to the full text.

## Layout (approved: "A — top blocks")

Single-column Tufte reading flow with the right margin reserved for sidenotes and
figure callouts. Top to bottom:

1. Title + authors/year/DOI subtitle.
2. **摘要 / Abstract** block (original abstract).
3. **目录 / Outline** block (parsed sections: heading + page).
4. v3 card sections (see template below).

## Components

### 1. `internal/apiclient/client.go` — v3 contract

Replace the `Card` struct and extend `PaperDetail`/`Figure`:

```go
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
type CardMethodology struct {
    Problem string `json:"problem"`
    Method  string `json:"method"`
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
```

`CardFigure` and `Evidence` are unchanged. Add to `PaperDetail`:
`Abstract *string json:"abstract,omitempty"` and
`Sections []Section json:"sections"` where
`Section{Order int32; Heading *string; PageStart *int32; PageEnd *int32}` (omit the
section body `text` — the outline does not need it). Add to `Figure`:
`ID string json:"id"` and `HasImage bool json:"has_image"`.

New method for the image proxy:

```go
// GetFigureImage streams the API's figure-image endpoint. The caller closes the
// returned reader. Returns ErrNotFound on a 404.
func (c *Client) GetFigureImage(ctx context.Context, paperID, figureID string) (io.ReadCloser, string, error)
```

It GETs `/v1/papers/{paperID}/figures/{figureID}/image`, returns the response body
and `Content-Type`, maps 404 → `ErrNotFound`, and other non-2xx → error (the caller
closes the body; do not `defer Close` it inside the method on the success path).

### 2. Image-proxy route — `cmd/web/main.go` + `internal/web/handlers.go`

The browser cannot reach the API directly, so the viewer proxies images. Add the
`GetFigureImage` method to the `API` interface and a handler:

```go
func (h *Handler) FigureImage(w http.ResponseWriter, r *http.Request) {
    paperID := chi.URLParam(r, "id")
    figureID := chi.URLParam(r, "figureId")
    body, contentType, err := h.api.GetFigureImage(r.Context(), paperID, figureID)
    if errors.Is(err, apiclient.ErrNotFound) { http.NotFound(w, r); return }
    if err != nil { h.renderError(w, http.StatusBadGateway, ...); return }
    defer body.Close()
    if contentType == "" { contentType = "image/png" }
    w.Header().Set("Content-Type", contentType)
    _, _ = io.Copy(w, body)
}
```

Route in `main.go`: `r.Get("/papers/{id}/figures/{figureId}/image", h.FigureImage)`.

### 3. `internal/web/view.go` — view model

- Add `Outline []OutlineEntry` to `PaperView`, where `OutlineEntry{Heading string;
  Page *int}`. Build it from `Detail.Sections`, skipping entries whose `Heading` is
  nil/empty; `Page` = `PageStart`.
- Extend `FigureNote` with `HasImage bool` and `ImageURL string`. In
  `BuildPaperView`, while grouping card figures, resolve each figure's label against
  a `map[normalizedLabel]Figure` built from `Detail.Figures` to obtain `ID`/`HasImage`;
  set `ImageURL = "/papers/" + Detail.PaperID + "/figures/" + figure.ID + "/image"`
  when `HasImage` (else empty). `Caption` resolution stays as today.
- No change for evidence truncation — the full `Snippet` already lives in
  `EvidenceNote`; truncation/hover is CSS-only.

Architecture-vs-chart is decided by anchor, not a heuristic: the template renders
the figures grouped under the `implementation` claim key inline, and all other
figure groups as margin thumbnails. `view.go` does not need an explicit flag.

### 4. `internal/web/templates/paper.tmpl`

Render order: title/subtitle → Abstract block → Outline block → v3 card sections:

- **引言** — `card.Introduction` (scalar), sidenotes/figures at `introduction`.
- **相关工作** — `card.RelatedWork` (scalar) at `related_work` (omit if empty).
- **方法** — `card.Methodology[]`: each item as "problem → method"; anchor
  `methodology#i`.
- **结果** — `card.Results[]`: each item shows `metric`, `finding`, a comparisons
  table (`work` / `value` / `reference`), and a `self_only` badge when true; anchor
  `results#i`; chart figures render as margin thumbnails here.
- **实现** — `card.Implementation.Overview` (anchor `implementation`, where the
  architecture diagram renders **inline**), then `Modules[]` each showing
  `name`/`function`/`design`/`principle`; anchor `modules#i`.
- **链接** — code/data links (unchanged).

Sub-templates:
- `sidenotes` reworked to the **expand-in-place** style: an always-visible inline
  superscript number tied to a margin note that shows the fixed `[§section]`/`[p.N]`
  prefix plus a one-line-truncated snippet; on hover the note reflows to full text.
  Drop the checkbox toggle.
- `figurenotes` (margin thumbnails): when `HasImage`, a thumbnail `<img
  src="ImageURL">` wrapped in a CSS-only enlarge control (checkbox/`:target`
  lightbox) plus the `label`/`page`/`caption`; when `!HasImage`, fall back to the
  current text caption callout.
- New `inlinefigure` for the implementation section: a full-width `<figure>` with
  the image and a `<figcaption>` (label + page), used for the architecture diagram;
  text-caption fallback when `!HasImage`.

### 5. `internal/web/static/tufte.css`

Add styles for: the abstract block and outline block; the expand-in-place truncated
sidenote (one-line clamp via `white-space:nowrap; overflow:hidden;
text-overflow:ellipsis`, expanding on `:hover`); the inline `<figure>` block; the
margin thumbnail and its CSS-only enlarge/lightbox; the results comparisons table;
and the `self_only` badge. Keep the existing Tufte variables/colors; no JS.

## Data flow

`GET /papers/{id}` (web) → `apiclient.GetPaper` → `PaperDetail` (now with abstract,
sections, figure id/has_image, v3 card) → `BuildPaperView` (outline + figure image
URLs + evidence/figure grouping) → `paper.tmpl`. Figure `<img>` tags point at the
web's own `/papers/{id}/figures/{figureId}/image`, which proxies the API.

## Error handling

- Missing card (`Card == nil`) → existing "reading not complete" notice; abstract +
  outline still render.
- Figure without `has_image` → text-caption fallback (no broken `<img>`).
- Image proxy: API 404 → 404; API error → 502 error page.
- Empty/old 2.0 cards: v3 fields are absent → their sections are omitted (templates
  guard with `with`/`if`); the abstract and outline still render.

## Testing

- `apiclient/client_test.go`: decode a v3 card + `abstract` + `sections` + figure
  `id`/`has_image`; `GetFigureImage` returns body + content-type and maps 404 →
  `ErrNotFound`.
- `web/view_test.go` (new): `BuildPaperView` builds the outline (skips empty
  headings, carries page), and resolves a figure's `ImageURL`/`HasImage` by label;
  an implementation-anchored figure groups under `implementation`.
- `web/handlers_test.go`: the image-proxy handler streams bytes with the API's
  content-type (200) and returns 404 when the API reports not-found (fake API).
- `web/render_test.go`: `paper.tmpl` renders the abstract block, the outline,
  the v3 sections (methodology pairs, results with a comparisons table + self_only
  badge, implementation modules), an `<img>` for a figure with an image, and the
  truncated-snippet sidenote markup.

## Known caveats

- Stored 2.0 cards do not populate the v3 sections; re-read the paper to get v3.
- The outline is flat (GROBID sections have no nesting in the DTO); headings + page
  numbers only.
- CSS-only enlarge/lightbox keeps the app JS-free, consistent with the current Tufte
  checkbox-toggle approach.
