# Collection Homepage Redesign

Date: 2026-06-29
Status: Approved (pre-implementation)

## Goal

Redesign the ScholarFlow paper-collection homepage (`scholarflow-web`) so that:

1. Papers are split into **per-source pages** — arXiv vs. manual (local) upload (the only two sources today).
2. Within each page, papers are organized into **blocks**, grouped by **date** or **category**, user-switchable.
3. The **paper status display** is fixed to render all backend states with friendly labels, not just `queued`/`parsed`.

## Context & Constraints

- `scholarflow-web` is **read-only** and never touches the DB; it consumes the server's REST API (`GET /v1/papers`, `GET /v1/papers/{id}`).
- The current homepage (`collection.tmpl`) is a single flat `<ul>` of all papers with a raw status badge and a `year · filename` meta line. No grouping, no source separation.
- The current `GET /v1/papers` response (`PaperSummary`) returns only `paper_id, title, status, publication_year, uploaded_filename, created_at`. It does **not** expose source or category.
- The DB already stores `source_type` (`'arxiv'` | `'local_pdf'`) and `source_id`. arXiv `primary_category` (e.g. `cs.CL`) is parsed by the source adapter (`sources.Entry.PrimaryCategory`) but **discarded** — not carried in `SourceInfo`, not persisted.
- The "status bug": the template prints the raw `.Status` string and the CSS styles all six states. Papers appear stuck at `queued`/`parsed` because the LLM reader is disabled by default, so they never advance to `reading`/`completed`. The fix is friendly label mapping for all states; no filtering exists to remove.

## Backend Changes (`scholarflow-server`)

Additive only — a nullable column and extra JSON fields. Nothing existing breaks.

### 1. Persist arXiv primary category

- Migration `migrations/00005_paper_primary_category.sql` (goose-annotated):
  - Up: `ALTER TABLE papers ADD COLUMN primary_category TEXT;`
  - Down: `ALTER TABLE papers DROP COLUMN primary_category;`
- `papers.SourceInfo` (internal/papers/models.go): add `PrimaryCategory string`.
- `queries/papers.sql` `CreatePaper`: add `primary_category` to the column list, params, and `RETURNING`.
- `internal/papers/repository.go` `CreatePaperUpload`: pass `PrimaryCategory: optString(info.PrimaryCategory)`.
- `internal/jobs/harvest_pipeline.go` `ingestEntry`: set `PrimaryCategory: e.PrimaryCategory` on the `SourceInfo`.
- Local uploads (`UploadPDF`) leave `PrimaryCategory` empty → column null.

### 2. Expose source & category in the list API

- `queries/papers.sql` `ListPapers`: add `source_type, source_id, primary_category` to the SELECT.
- `internal/papers/read.go` `PaperSummary`: add
  - `SourceType string \`json:"source_type"\``
  - `SourceID *string \`json:"source_id,omitempty"\``
  - `PrimaryCategory *string \`json:"primary_category,omitempty"\``
  - and map them in `ListPapers`.
- Run `sqlc generate` (do not hand-edit `*.sql.go`).

### 3. Tests

- Harvest pipeline test asserts `PrimaryCategory` is threaded into the ingested `SourceInfo`.
- (Existing service tests already cover local vs arxiv source; extend assertions where cheap.)

## Web Changes (`scholarflow-web`)

### Routing

Keep a **single** collection route to avoid a chi conflict with `/papers/{id}`:

- `GET /?source=<arxiv|local>&group=<date|category>`
  - `source` default `arxiv`; unknown values fall back to `arxiv`.
  - `group` default `date`; unknown values fall back to `date`.
  - `source=local` selects `source_type == "local_pdf"`.
- `GET /papers/{id}` and `GET /papers/{id}/figures/{figureId}/image` unchanged.

### API client

`internal/apiclient/client.go` `PaperSummary`: add `SourceType string`, `SourceID *string`, `PrimaryCategory *string` matching the new JSON.

### View model (new file `internal/web/collection_view.go`)

`BuildCollectionView(summaries []apiclient.PaperSummary, source, group string) CollectionView` produces:

```
CollectionView {
    Source       string              // "arxiv" | "local" (normalized)
    Group        string              // "date" | "category" (normalized)
    Counts       struct{ Arxiv, Local int }  // for nav-tab counts (both sources)
    Blocks       []CollectionBlock
}
CollectionBlock {
    Label  string          // e.g. "2024", "cs.CL", "未知年份", "未分类"
    Papers []PaperRow
}
PaperRow {
    PaperID     string
    Title       string    // Title, else UploadedFilename
    Filename    string
    Year        *int32
    StatusLabel string    // friendly label
    StatusClass string    // existing CSS status-<state> class
}
```

Behavior:
- Filter `summaries` by selected source (`arxiv` → `source_type == "arxiv"`; `local` → `source_type == "local_pdf"`). `Counts` is computed over the full unfiltered list so both tabs show totals.
- **group=date:** bucket by `publication_year` (int). Sort blocks by year descending. Papers without a year go to a `未知年份` block placed last.
- **group=category:** bucket by `primary_category` (string). Sort blocks by paper count descending, then label ascending for stable ties. Empty/null category → `未分类` block placed last.
- Within a block, papers sorted by title (stable; falls back to filename).
- Status label/class mapping (single source of truth, table below).

Status mapping:

| backend status | label | class |
|---|---|---|
| queued | 排队中 | status-queued |
| processing | 解析中 | status-processing |
| parsed | 已解析 | status-parsed |
| reading | 阅读中 | status-reading |
| completed | 已完成 | status-completed |
| failed | 失败 | status-failed |
| (unknown) | the raw status | status-queued |

### Handler

`internal/web/handlers.go` `Collection`: read `source`/`group` query params, call `ListPapers`, build the view via `BuildCollectionView`, render `collection.tmpl`. Same backend-unavailable error path as today.

### Template (`internal/web/templates/collection.tmpl`)

- Header: title + subtitle.
- **Nav tabs**: two links (`arXiv (n)`, `本地上传 (n)`) preserving the current `group`; active tab marked.
- **Group toggle**: two links (`按日期`, `按类别`) preserving the current `source`; active marked.
- **Blocks**: for each block, a `<section>` with a `.block-label` header (label + count) and a `.paper-list` of rows. Each row: title link → `/papers/{id}`, status badge, meta (`year · filename`).
- Empty state: existing `还没有论文` notice when the selected source has no papers.

### CSS (`internal/web/static/app.css`)

Add, reusing existing tokens:
- `.coll-nav` / `.coll-tab` (+ active state) for source tabs.
- `.coll-toggle` / `.coll-toggle a` (+ active) for the date/category switch.
- `.coll-block` section spacing; reuse `.block-label` for headers, with a count suffix.
- Keep existing `.paper-list`, `.status-badge`, `.status-*` rules.

### Tests

- `collection_view_test.go`: grouping by date (descending, unknown-year bucket last), grouping by category (count-desc, uncategorized last), source filtering, counts over full list, status label/class mapping for all six + unknown.
- Extend `handlers_test.go`: query-param parsing/defaults and that the selected source/group reach the rendered output.

## Non-Goals (YAGNI)

- No new API endpoint; reuse `GET /v1/papers`.
- No multi-category support — `primary_category` only (single value per arXiv paper).
- No client-side JS, search, pagination, or sorting controls beyond the two toggles.
- No date grouping by month/upload-date — `publication_year` only.

## Risks / Notes

- Local uploads always land in `未分类` under category grouping; that's expected (no category data for them).
- `source_id` is exposed but not yet surfaced in the UI; included now so a future "view on arXiv" link needs no further API change.
