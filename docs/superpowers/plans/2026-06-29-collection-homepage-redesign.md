# Collection Homepage Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the ScholarFlow web collection homepage into per-source pages (arXiv / manual upload) with date- or category-grouped blocks, and fix the paper status display to cover all backend states.

**Architecture:** Additive backend changes expose `source_type`, `source_id`, and a new `primary_category` column via the existing `GET /v1/papers` endpoint. The read-only web module groups and labels papers in a new view-model file, rendered server-side via a rewritten template; navigation and grouping are driven by `?source=&group=` query params (no JS).

**Tech Stack:** Go, chi router, `html/template`, sqlc (pgx/v5), goose migrations, PostgreSQL.

## Global Constraints

- Two **separate** git repos: `scholarflow-server/` and `scholarflow-web/`. Each task names which repo it is in. Commit within that repo.
- Web module is **read-only** — it never touches the DB, only the server REST API.
- Backend DB layer is sqlc-generated: **do not hand-edit `internal/db/*.sql.go`**. Edit `queries/papers.sql` + `migrations/`, then run sqlc. sqlc invocation (from `scholarflow-server/`): `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` (sqlc is not on PATH).
- Backend status values are exactly: `queued`, `processing`, `parsed`, `reading`, `completed`, `failed`.
- Source types are exactly: `arxiv`, `local_pdf`.
- UI copy is Chinese (matches the existing templates).
- Manual uploads have no category → they fall into a single `未分类` block under category grouping. Date grouping uses `publication_year` only; papers without a year fall into `未知年份` (placed last).
- Go style: `go fmt`; tests use stdlib `testing`, co-located `*_test.go`.

---

### Task 1: Persist arXiv primary category through ingestion (server)

**Repo:** `scholarflow-server/`

**Files:**
- Create: `migrations/00005_paper_primary_category.sql`
- Modify: `queries/papers.sql` (`CreatePaper`)
- Modify: `internal/papers/models.go` (`SourceInfo`)
- Modify: `internal/papers/repository.go` (`CreatePaperUpload`)
- Modify: `internal/jobs/harvest_pipeline.go` (`ingestEntry`)
- Regenerate: `internal/db/*` (sqlc)
- Test: `internal/jobs/harvest_pipeline_test.go`

**Interfaces:**
- Consumes: `sources.Entry.PrimaryCategory string` (already parsed by the arXiv adapter).
- Produces: `papers.SourceInfo.PrimaryCategory string`; DB column `papers.primary_category TEXT` (nullable); `db.CreatePaperParams.PrimaryCategory *string`.

- [ ] **Step 1: Write the failing test**

Add this test to `internal/jobs/harvest_pipeline_test.go`:

```go
func TestHarvestThreadsPrimaryCategory(t *testing.T) {
	src := &fakeSource{name: "arxiv", entries: map[string][]sources.Entry{
		"cs.CL": {
			{SourceID: "2301.00009", Title: "Cat", PDFURL: "u9", PrimaryCategory: "cs.CL",
				Published: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)},
		},
	}}
	ing := &fakeIngester{existing: map[string]bool{}}
	fetch := &fakeFetcher{data: map[string][]byte{"u9": []byte("%PDF-1.4 content")}}
	h := NewHarvestPipeline([]sources.Source{src}, []string{"cs.CL"}, 25, 0, ing, fetch)

	if err := h.Harvest(context.Background(), nil); err != nil {
		t.Fatalf("Harvest error: %v", err)
	}
	if len(ing.ingested) != 1 || ing.ingested[0].PrimaryCategory != "cs.CL" {
		t.Fatalf("PrimaryCategory = %#v, want cs.CL", ing.ingested)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/jobs/ -run TestHarvestThreadsPrimaryCategory`
Expected: FAIL — compile error `info.PrimaryCategory undefined (type papers.SourceInfo has no field PrimaryCategory)`.

- [ ] **Step 3: Add the migration**

Create `migrations/00005_paper_primary_category.sql`:

```sql
-- +goose Up
ALTER TABLE papers ADD COLUMN primary_category TEXT;

-- +goose Down
ALTER TABLE papers DROP COLUMN primary_category;
```

- [ ] **Step 4: Add the field to `SourceInfo`**

In `internal/papers/models.go`, add `PrimaryCategory` to the `SourceInfo` struct:

```go
type SourceInfo struct {
	SourceType      string
	SourceID        string
	Filename        string
	Title           string
	Abstract        string
	DOI             string
	Year            int32
	PrimaryCategory string
}
```

- [ ] **Step 5: Add the column to the `CreatePaper` query**

In `queries/papers.sql`, replace the `CreatePaper` block with:

```sql
-- name: CreatePaper :one
INSERT INTO papers (source_type, source_id, status, uploaded_filename, title, abstract, doi, publication_year, primary_category)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;
```

- [ ] **Step 6: Regenerate sqlc**

Run (from `scholarflow-server/`): `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`
Expected: `internal/db/models.go` `Paper` struct gains `PrimaryCategory *string`; `internal/db/papers.sql.go` `CreatePaperParams` gains `PrimaryCategory *string` and the `createPaper` SQL/scan include the new column. Do not edit these files by hand.

- [ ] **Step 7: Pass the field in the repository**

In `internal/papers/repository.go`, in `CreatePaperUpload`'s `db.CreatePaperParams{...}`, add the line after `PublicationYear`:

```go
		PublicationYear:  optInt32(info.Year),
		PrimaryCategory:  optString(info.PrimaryCategory),
```

- [ ] **Step 8: Thread the category in the harvest pipeline**

In `internal/jobs/harvest_pipeline.go`, in `ingestEntry`, add `PrimaryCategory` to the `papers.SourceInfo{...}` literal:

```go
	info := papers.SourceInfo{
		SourceType:      src.Name(),
		SourceID:        e.SourceID,
		Filename:        filenameForEntry(e),
		Title:           e.Title,
		Abstract:        e.Abstract,
		DOI:             e.DOI,
		Year:            year,
		PrimaryCategory: e.PrimaryCategory,
	}
```

- [ ] **Step 9: Run the test to verify it passes**

Run: `go test ./internal/jobs/ -run TestHarvestThreadsPrimaryCategory`
Expected: PASS

- [ ] **Step 10: Run the full package tests + fmt**

Run: `go fmt ./... && go test ./...`
Expected: PASS (all existing tests still green).

- [ ] **Step 11: Commit**

```bash
git add migrations/00005_paper_primary_category.sql queries/papers.sql internal/papers/models.go internal/papers/repository.go internal/jobs/harvest_pipeline.go internal/jobs/harvest_pipeline_test.go internal/db
git commit -m "feat(server): persist arxiv primary_category through ingestion"
```

---

### Task 2: Expose source & category in the list API (server)

**Repo:** `scholarflow-server/`

**Files:**
- Modify: `queries/papers.sql` (`ListPapers`)
- Regenerate: `internal/db/*` (sqlc)
- Modify: `internal/papers/read.go` (`PaperSummary` + `ListPapers` mapping)
- Test: `internal/papers/read_test.go` (create)

**Interfaces:**
- Consumes: `db.ListPapersRow` (gains `SourceType string`, `SourceID *string`, `PrimaryCategory *string` after sqlc regen).
- Produces: JSON fields `source_type` (string), `source_id` (string, omitempty), `primary_category` (string, omitempty) on each `GET /v1/papers` element.

- [ ] **Step 1: Write the failing test**

Create `internal/papers/read_test.go`:

```go
package papers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPaperSummaryMarshalsSourceFields(t *testing.T) {
	sid := "2301.00001"
	cat := "cs.CL"
	b, err := json.Marshal(PaperSummary{
		Status:          "completed",
		SourceType:      "arxiv",
		SourceID:        &sid,
		PrimaryCategory: &cat,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := string(b)
	for _, want := range []string{`"source_type":"arxiv"`, `"source_id":"2301.00001"`, `"primary_category":"cs.CL"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/papers/ -run TestPaperSummaryMarshalsSourceFields`
Expected: FAIL — compile error `unknown field SourceType in struct literal of type PaperSummary`.

- [ ] **Step 3: Add fields to `PaperSummary`**

In `internal/papers/read.go`, replace the `PaperSummary` struct with:

```go
type PaperSummary struct {
	PaperID          uuid.UUID  `json:"paper_id"`
	Title            *string    `json:"title,omitempty"`
	Status           string     `json:"status"`
	PublicationYear  *int32     `json:"publication_year,omitempty"`
	UploadedFilename string     `json:"uploaded_filename"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	SourceType       string     `json:"source_type"`
	SourceID         *string    `json:"source_id,omitempty"`
	PrimaryCategory  *string    `json:"primary_category,omitempty"`
}
```

- [ ] **Step 4: Add columns to the `ListPapers` query**

In `queries/papers.sql`, replace the `ListPapers` SELECT with:

```sql
-- name: ListPapers :many
SELECT id, title, status, publication_year, uploaded_filename, created_at, source_type, source_id, primary_category
FROM papers
ORDER BY created_at DESC
LIMIT 500;
```

- [ ] **Step 5: Regenerate sqlc**

Run (from `scholarflow-server/`): `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`
Expected: `internal/db/papers.sql.go` `ListPapersRow` gains `SourceType string`, `SourceID *string`, `PrimaryCategory *string`, and the scan reads them. Do not edit by hand.

- [ ] **Step 6: Map the new columns**

In `internal/papers/read.go`, in `ListPapers`, extend the `PaperSummary{...}` built per row:

```go
		summaries = append(summaries, PaperSummary{
			PaperID:          row.ID,
			Title:            row.Title,
			Status:           row.Status,
			PublicationYear:  row.PublicationYear,
			UploadedFilename: row.UploadedFilename,
			CreatedAt:        timestamp(row.CreatedAt),
			SourceType:       row.SourceType,
			SourceID:         row.SourceID,
			PrimaryCategory:  row.PrimaryCategory,
		})
```

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/papers/ -run TestPaperSummaryMarshalsSourceFields`
Expected: PASS

- [ ] **Step 8: Run full tests + fmt**

Run: `go fmt ./... && go test ./...`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add queries/papers.sql internal/papers/read.go internal/papers/read_test.go internal/db
git commit -m "feat(server): expose source_type/source_id/primary_category in list API"
```

---

### Task 3: Collection view model + status mapping (web)

**Repo:** `scholarflow-web/`

**Files:**
- Modify: `internal/apiclient/client.go` (`PaperSummary`)
- Create: `internal/web/collection_view.go`
- Test: `internal/web/collection_view_test.go`

**Interfaces:**
- Consumes: `apiclient.PaperSummary` (gains `SourceType string`, `SourceID *string`, `PrimaryCategory *string`).
- Produces:
  - `func BuildCollectionView(summaries []apiclient.PaperSummary, source, group string) CollectionView`
  - Types `CollectionView{Source string; Group string; Counts SourceCounts; Blocks []CollectionBlock}`, `SourceCounts{Arxiv int; Local int}`, `CollectionBlock{Label string; Papers []PaperRow}`, `PaperRow{PaperID, Title, Filename string; Year *int32; StatusLabel, StatusClass string}`.
  - `func statusDisplay(status string) (label, class string)`

- [ ] **Step 1: Add fields to the API client `PaperSummary`**

In `internal/apiclient/client.go`, replace the `PaperSummary` struct with:

```go
type PaperSummary struct {
	PaperID          string     `json:"paper_id"`
	Title            *string    `json:"title,omitempty"`
	Status           string     `json:"status"`
	PublicationYear  *int32     `json:"publication_year,omitempty"`
	UploadedFilename string     `json:"uploaded_filename"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	SourceType       string     `json:"source_type"`
	SourceID         *string    `json:"source_id,omitempty"`
	PrimaryCategory  *string    `json:"primary_category,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests**

Create `internal/web/collection_view_test.go`:

```go
package web

import (
	"testing"

	"scholarflow_web/internal/apiclient"
)

func i32(v int32) *int32 { return &v }
func sp(v string) *string { return &v }

func arxiv(id, status string, year *int32, cat *string) apiclient.PaperSummary {
	return apiclient.PaperSummary{PaperID: id, Status: status, UploadedFilename: id + ".pdf",
		SourceType: "arxiv", PublicationYear: year, PrimaryCategory: cat}
}

func local(id, status string, year *int32) apiclient.PaperSummary {
	return apiclient.PaperSummary{PaperID: id, Status: status, UploadedFilename: id + ".pdf",
		SourceType: "local_pdf", PublicationYear: year}
}

func TestBuildCollectionCountsBothSources(t *testing.T) {
	v := BuildCollectionView([]apiclient.PaperSummary{
		arxiv("a1", "parsed", i32(2024), sp("cs.CL")),
		arxiv("a2", "parsed", i32(2023), sp("cs.LG")),
		local("l1", "queued", nil),
	}, "arxiv", "date")
	if v.Counts.Arxiv != 2 || v.Counts.Local != 1 {
		t.Fatalf("counts = %+v", v.Counts)
	}
}

func TestBuildCollectionFiltersBySource(t *testing.T) {
	v := BuildCollectionView([]apiclient.PaperSummary{
		arxiv("a1", "parsed", i32(2024), sp("cs.CL")),
		local("l1", "queued", nil),
	}, "local", "date")
	if v.Source != "local" {
		t.Fatalf("source = %q", v.Source)
	}
	total := 0
	for _, b := range v.Blocks {
		total += len(b.Papers)
	}
	if total != 1 || v.Blocks[0].Papers[0].PaperID != "l1" {
		t.Fatalf("blocks = %+v", v.Blocks)
	}
}

func TestGroupByDateDescendingUnknownLast(t *testing.T) {
	v := BuildCollectionView([]apiclient.PaperSummary{
		arxiv("a1", "parsed", i32(2023), nil),
		arxiv("a2", "parsed", i32(2024), nil),
		arxiv("a3", "parsed", nil, nil),
	}, "arxiv", "date")
	if len(v.Blocks) != 3 {
		t.Fatalf("blocks = %d", len(v.Blocks))
	}
	if v.Blocks[0].Label != "2024" || v.Blocks[1].Label != "2023" || v.Blocks[2].Label != "未知年份" {
		t.Fatalf("labels = %q %q %q", v.Blocks[0].Label, v.Blocks[1].Label, v.Blocks[2].Label)
	}
}

func TestGroupByCategoryCountDescUncategorizedLast(t *testing.T) {
	v := BuildCollectionView([]apiclient.PaperSummary{
		arxiv("a1", "parsed", i32(2024), sp("cs.LG")),
		arxiv("a2", "parsed", i32(2024), sp("cs.CL")),
		arxiv("a3", "parsed", i32(2024), sp("cs.CL")),
		arxiv("a4", "parsed", i32(2024), nil),
	}, "arxiv", "category")
	if len(v.Blocks) != 3 {
		t.Fatalf("blocks = %d", len(v.Blocks))
	}
	if v.Blocks[0].Label != "cs.CL" || v.Blocks[1].Label != "cs.LG" || v.Blocks[2].Label != "未分类" {
		t.Fatalf("labels = %q %q %q", v.Blocks[0].Label, v.Blocks[1].Label, v.Blocks[2].Label)
	}
}

func TestNormalizeDefaults(t *testing.T) {
	v := BuildCollectionView(nil, "bogus", "bogus")
	if v.Source != "arxiv" || v.Group != "date" {
		t.Fatalf("got source=%q group=%q", v.Source, v.Group)
	}
}

func TestStatusDisplayAll(t *testing.T) {
	cases := map[string][2]string{
		"queued":     {"排队中", "status-queued"},
		"processing": {"解析中", "status-processing"},
		"parsed":     {"已解析", "status-parsed"},
		"reading":    {"阅读中", "status-reading"},
		"completed":  {"已完成", "status-completed"},
		"failed":     {"失败", "status-failed"},
		"weird":      {"weird", "status-queued"},
	}
	for status, want := range cases {
		label, class := statusDisplay(status)
		if label != want[0] || class != want[1] {
			t.Fatalf("%s -> %q/%q, want %q/%q", status, label, class, want[0], want[1])
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/web/ -run 'Collection|Group|Normalize|StatusDisplay'`
Expected: FAIL — `undefined: BuildCollectionView` / `undefined: statusDisplay`.

- [ ] **Step 4: Write the view model**

Create `internal/web/collection_view.go`:

```go
package web

import (
	"sort"
	"strconv"

	"scholarflow_web/internal/apiclient"
)

const (
	sourceArxiv = "arxiv"
	sourceLocal = "local"

	groupDate     = "date"
	groupCategory = "category"

	sourceTypeArxiv = "arxiv"
	sourceTypeLocal = "local_pdf"

	labelUnknownYear   = "未知年份"
	labelUncategorized = "未分类"
)

// CollectionView is the homepage model: the selected source/group, per-source
// counts for the nav tabs, and the papers organized into display blocks.
type CollectionView struct {
	Source string
	Group  string
	Counts SourceCounts
	Blocks []CollectionBlock
}

type SourceCounts struct {
	Arxiv int
	Local int
}

type CollectionBlock struct {
	Label  string
	Papers []PaperRow
}

type PaperRow struct {
	PaperID     string
	Title       string
	Filename    string
	Year        *int32
	StatusLabel string
	StatusClass string
}

// BuildCollectionView filters summaries to the selected source, groups them by
// date or category, and maps statuses to display labels. source/group are
// normalized (defaults: arxiv/date). Counts cover all sources, not just the
// selected one, so both nav tabs show totals.
func BuildCollectionView(summaries []apiclient.PaperSummary, source, group string) CollectionView {
	source = normalizeSource(source)
	group = normalizeGroup(group)
	view := CollectionView{Source: source, Group: group}

	wantType := sourceTypeArxiv
	if source == sourceLocal {
		wantType = sourceTypeLocal
	}

	var rows []apiclient.PaperSummary
	for _, s := range summaries {
		switch s.SourceType {
		case sourceTypeArxiv:
			view.Counts.Arxiv++
		case sourceTypeLocal:
			view.Counts.Local++
		}
		if s.SourceType == wantType {
			rows = append(rows, s)
		}
	}

	if group == groupCategory {
		view.Blocks = groupByCategory(rows)
	} else {
		view.Blocks = groupByDate(rows)
	}
	return view
}

func normalizeSource(s string) string {
	if s == sourceLocal {
		return sourceLocal
	}
	return sourceArxiv
}

func normalizeGroup(g string) string {
	if g == groupCategory {
		return groupCategory
	}
	return groupDate
}

func toRow(s apiclient.PaperSummary) PaperRow {
	title := s.UploadedFilename
	if s.Title != nil && *s.Title != "" {
		title = *s.Title
	}
	label, class := statusDisplay(s.Status)
	return PaperRow{
		PaperID:     s.PaperID,
		Title:       title,
		Filename:    s.UploadedFilename,
		Year:        s.PublicationYear,
		StatusLabel: label,
		StatusClass: class,
	}
}

func groupByDate(rows []apiclient.PaperSummary) []CollectionBlock {
	buckets := map[string][]PaperRow{}
	var years []int
	seen := map[int]bool{}
	for _, s := range rows {
		key := labelUnknownYear
		if s.PublicationYear != nil {
			y := int(*s.PublicationYear)
			key = strconv.Itoa(y)
			if !seen[y] {
				seen[y] = true
				years = append(years, y)
			}
		}
		buckets[key] = append(buckets[key], toRow(s))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	var blocks []CollectionBlock
	for _, y := range years {
		key := strconv.Itoa(y)
		blocks = append(blocks, CollectionBlock{Label: key, Papers: sortRows(buckets[key])})
	}
	if rs, ok := buckets[labelUnknownYear]; ok {
		blocks = append(blocks, CollectionBlock{Label: labelUnknownYear, Papers: sortRows(rs)})
	}
	return blocks
}

func groupByCategory(rows []apiclient.PaperSummary) []CollectionBlock {
	buckets := map[string][]PaperRow{}
	for _, s := range rows {
		key := labelUncategorized
		if s.PrimaryCategory != nil && *s.PrimaryCategory != "" {
			key = *s.PrimaryCategory
		}
		buckets[key] = append(buckets[key], toRow(s))
	}
	var labels []string
	for k := range buckets {
		if k != labelUncategorized {
			labels = append(labels, k)
		}
	}
	sort.Slice(labels, func(i, j int) bool {
		if len(buckets[labels[i]]) != len(buckets[labels[j]]) {
			return len(buckets[labels[i]]) > len(buckets[labels[j]])
		}
		return labels[i] < labels[j]
	})
	var blocks []CollectionBlock
	for _, k := range labels {
		blocks = append(blocks, CollectionBlock{Label: k, Papers: sortRows(buckets[k])})
	}
	if rs, ok := buckets[labelUncategorized]; ok {
		blocks = append(blocks, CollectionBlock{Label: labelUncategorized, Papers: sortRows(rs)})
	}
	return blocks
}

func sortRows(rows []PaperRow) []PaperRow {
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Title < rows[j].Title
	})
	return rows
}

// statusDisplay maps a backend status to a Chinese label and the existing CSS
// status class. Unknown statuses render their raw text with a neutral class.
func statusDisplay(status string) (label, class string) {
	switch status {
	case "queued":
		return "排队中", "status-queued"
	case "processing":
		return "解析中", "status-processing"
	case "parsed":
		return "已解析", "status-parsed"
	case "reading":
		return "阅读中", "status-reading"
	case "completed":
		return "已完成", "status-completed"
	case "failed":
		return "失败", "status-failed"
	default:
		return status, "status-queued"
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/web/ -run 'Collection|Group|Normalize|StatusDisplay'`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/apiclient/client.go internal/web/collection_view.go internal/web/collection_view_test.go
git commit -m "feat(web): collection view model with source/date/category grouping"
```

---

### Task 4: Handler, template & CSS wiring (web)

**Repo:** `scholarflow-web/`

**Files:**
- Modify: `internal/web/handlers.go` (`Collection`)
- Modify: `internal/web/templates/collection.tmpl`
- Modify: `internal/web/static/app.css`
- Modify: `internal/web/render_test.go` (two existing tests pass new model)
- Modify: `internal/web/handlers_test.go` (existing row test + new param test)

**Interfaces:**
- Consumes: `BuildCollectionView`, `CollectionView` (Task 3).
- Produces: rendered `/` page; `collection.tmpl` now takes a `CollectionView` (not `[]apiclient.PaperSummary`).

- [ ] **Step 1: Update existing tests to the new model (these are the failing tests)**

In `internal/web/render_test.go`, replace the body of `TestBaseUsesEditorialStylesheetAndKatex` line that calls Render:

```go
	if err := Render(&b, "collection.tmpl", CollectionView{}); err != nil {
```

In the same file, replace `TestRenderCollection` with:

```go
func TestRenderCollection(t *testing.T) {
	title := "深度学习论文"
	var b strings.Builder
	view := BuildCollectionView([]apiclient.PaperSummary{
		{PaperID: "p1", Title: &title, Status: "completed", UploadedFilename: "a.pdf", SourceType: "arxiv"},
	}, "arxiv", "date")
	if err := Render(&b, "collection.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{"深度学习论文", "/papers/p1", "status-completed", "已完成", "coll-tab", "按类别"} {
		if !strings.Contains(out, want) {
			t.Fatalf("collection missing %q in:\n%s", want, out)
		}
	}
}
```

In `internal/web/handlers_test.go`, in `TestCollectionRendersRows`, add `SourceType: "arxiv"` to the summary literal (otherwise it is filtered out of the default arxiv view):

```go
	routerFor(&fakeAPI{summaries: []apiclient.PaperSummary{{PaperID: "p1", Title: &title, Status: "completed", UploadedFilename: "a.pdf", SourceType: "arxiv"}}}).
```

Also add a new test to `internal/web/handlers_test.go`:

```go
func TestCollectionSourceFilter(t *testing.T) {
	title := "本地论文"
	rr := httptest.NewRecorder()
	routerFor(&fakeAPI{summaries: []apiclient.PaperSummary{
		{PaperID: "ax", Status: "parsed", UploadedFilename: "ax.pdf", SourceType: "arxiv"},
		{PaperID: "lo", Title: &title, Status: "parsed", UploadedFilename: "lo.pdf", SourceType: "local_pdf"},
	}}).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/?source=local&group=date", nil))
	body := rr.Body.String()
	if !strings.Contains(body, "/papers/lo") || strings.Contains(body, "/papers/ax") {
		t.Fatalf("local view should show lo not ax:\n%s", body)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/web/ -run 'RenderCollection|BaseUsesEditorial|CollectionRendersRows|CollectionSourceFilter'`
Expected: FAIL — template still expects the old data shape / new assertions (`coll-tab`, source filter) not present yet.

- [ ] **Step 3: Update the handler**

In `internal/web/handlers.go`, replace the `Collection` method body:

```go
func (h *Handler) Collection(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.api.ListPapers(r.Context())
	if err != nil {
		h.renderError(w, http.StatusBadGateway, "后端不可用", "无法从 API 获取论文列表。")
		return
	}
	q := r.URL.Query()
	view := BuildCollectionView(summaries, q.Get("source"), q.Get("group"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := Render(w, "collection.tmpl", view); err != nil {
		log.Printf("render collection: %v", err)
	}
}
```

- [ ] **Step 4: Rewrite the collection template**

Replace the entire contents of `internal/web/templates/collection.tmpl`:

```html
{{define "title"}}ScholarFlow — 论文集{{end}}
{{define "content"}}
<h1>ScholarFlow</h1>
<p class="subtitle">已采集与阅读的论文</p>

<nav class="coll-nav">
  <a class="coll-tab{{if eq .Source "arxiv"}} active{{end}}" href="/?source=arxiv&group={{.Group}}">arXiv ({{.Counts.Arxiv}})</a>
  <a class="coll-tab{{if eq .Source "local"}} active{{end}}" href="/?source=local&group={{.Group}}">本地上传 ({{.Counts.Local}})</a>
</nav>

<div class="coll-toggle">
  <span>分组：</span>
  <a class="{{if eq .Group "date"}}active{{end}}" href="/?source={{.Source}}&group=date">按日期</a>
  <a class="{{if eq .Group "category"}}active{{end}}" href="/?source={{.Source}}&group=category">按类别</a>
</div>

{{if .Blocks}}
{{range .Blocks}}
<section class="coll-block">
  <span class="block-label">{{.Label}} · {{len .Papers}}</span>
  <ul class="paper-list">
  {{range .Papers}}
    <li>
      <a href="/papers/{{.PaperID}}">{{.Title}}</a>
      <span class="status-badge {{.StatusClass}}">{{.StatusLabel}}</span>
      <div class="paper-meta">{{if .Year}}{{.Year}} · {{end}}{{.Filename}}</div>
    </li>
  {{end}}
  </ul>
</section>
{{end}}
{{else}}
<p class="notice">这个来源下还没有论文。先通过 API 上传或采集一篇 PDF。</p>
{{end}}
{{end}}
```

- [ ] **Step 5: Add the CSS**

In `internal/web/static/app.css`, after the `/* collection + error pages */` block (the line ending `...border-radius:.3rem}` for `.notice`), append:

```css
/* collection nav tabs + grouping toggle + blocks */
.coll-nav{display:flex;gap:.3rem;border-bottom:1px solid var(--rule);margin:0 0 1rem}
.coll-tab{text-decoration:none;font-weight:600;padding:.4rem .8rem;color:var(--muted);border-bottom:2px solid transparent;margin-bottom:-1px}
.coll-tab.active{color:var(--accent-ink);border-bottom-color:var(--accent)}
.coll-toggle{display:flex;align-items:center;gap:.4rem;font-size:.82rem;color:var(--muted);margin:0 0 1.8rem}
.coll-toggle a{text-decoration:none;color:var(--muted);padding:.1rem .55rem;border:1px solid var(--rule);border-radius:.3rem}
.coll-toggle a.active{color:var(--accent-ink);border-color:var(--accent)}
.coll-block{margin:0 0 1.8rem}
.coll-block .block-label{border-bottom:1px solid var(--rule);padding-bottom:.25rem;margin-bottom:.5rem}
```

- [ ] **Step 6: Run the web tests to verify they pass**

Run: `go test ./...`
Expected: PASS (updated render/handler tests green, view tests still green).

- [ ] **Step 7: Format**

Run: `go fmt ./...`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add internal/web/handlers.go internal/web/templates/collection.tmpl internal/web/static/app.css internal/web/render_test.go internal/web/handlers_test.go
git commit -m "feat(web): per-source collection homepage with grouped blocks and status labels"
```

---

## Self-Review Notes

- **Spec coverage:** source split → Tasks 1/2 (data) + Task 4 (tabs); block grouping by date/category → Task 3 (`groupByDate`/`groupByCategory`) + Task 4 (toggle/template); status fix → Task 3 (`statusDisplay`, all six states) + Task 4 (template). `source_id` exposed (Task 2) though unused in UI, per spec note.
- **Manual `未分类`:** covered by `groupByCategory` empty-key bucket (Task 3) and confirmed by the user.
- **Type consistency:** `CollectionView`/`PaperRow`/`statusDisplay` names identical across Tasks 3 and 4; `SourceInfo.PrimaryCategory` / `CreatePaperParams.PrimaryCategory` / `ListPapersRow.PrimaryCategory` consistent across Tasks 1–2.
- **No DB-dependent unit tests:** backend tests use struct marshaling and fakes; web tests use in-memory fakes — no Docker needed.
```
