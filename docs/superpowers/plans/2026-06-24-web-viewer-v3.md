# Web Viewer v3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the read-only Tufte HTML viewer (`scholarflow-web`) to render the v3 paper card and add four reader features: the thesis outline, the original abstract, extracted figure images (architecture diagram inline, charts as margin thumbnails), and truncated-with-hover evidence comments.

**Architecture:** Three layered tasks on the server-side-rendered Go-template app (no client JS; CSS-only interactivity). Task 1 migrates the data layer (`apiclient`) and the reading template to v3 and applies the comment-truncation CSS. Task 2 adds the abstract + outline blocks. Task 3 adds figure images via a web-side image proxy. Custom styles go in `static/app.css` (loaded after `tufte.css`); the vendored `tufte.css` is left untouched.

**Tech Stack:** Go (`scholarflow_web`), chi router, `html/template` (embedded), Tufte CSS, standard `testing`.

## Global Constraints

- Module path `scholarflow_web`; run `go fmt ./...` before committing.
- Web-only slice: NO server (`scholarflow-server`) change. The API already returns `abstract`, `sections`, and figures with `id`/`has_image`.
- No client-side JavaScript. All interactivity is CSS-only (the app already uses the Tufte checkbox hack).
- Custom CSS goes in `internal/web/static/app.css` (loaded after `tufte.css`); do not edit `tufte.css`.
- Evidence comments must always show the sequence number (the Tufte counter), the section `[§N]`, and the page `[p.N]` when present; only the snippet truncates; hover expands in place.
- Architecture-vs-chart is decided by anchor: the figure(s) grouped under `claim_key:"implementation"` render inline; all others render as margin thumbnails. No image-content heuristic.
- Figures without `has_image` fall back to the existing text caption callout (no broken `<img>`).
- Tests use the standard `testing` package, co-located `*_test.go`, no Docker/network.

---

### Task 1: v3 data layer + reading template + comment truncation

**Files:**
- Modify: `internal/apiclient/client.go` (v3 `Card`, new `PaperDetail`/`Figure` fields, `Section` type)
- Modify: `internal/apiclient/client_test.go` (v3 decode test)
- Modify: `internal/web/templates/paper.tmpl` (v3 sections; `sidenotes` shows §+page)
- Modify: `internal/web/static/app.css` (results table, self_only badge, methodology/module styles, comment clamp+hover)
- Modify: `internal/web/render_test.go` (rewrite for v3)
- Modify: `internal/web/handlers_test.go` (v3 card literal)

**Interfaces:**
- Produces (consumed by Tasks 2 & 3):
  - `apiclient.Card{Introduction, RelatedWork string; Methodology []CardMethodology; Results []CardResult; Implementation CardImplementation; CodeLinks, DataLinks []string; Figures []CardFigure; Evidence []Evidence}`
  - `CardMethodology{Problem, Method string}`, `CardComparison{Work, Value, Reference string}`, `CardResult{Metric, Finding string; Comparisons []CardComparison; SelfOnly bool}`, `CardModule{Name, Function, Design, Principle string}`, `CardImplementation{Overview string; Modules []CardModule}`
  - `apiclient.PaperDetail` additionally has `Abstract *string` and `Sections []Section`; `Section{Order int32; Heading *string; PageStart *int32; PageEnd *int32}`
  - `apiclient.Figure` additionally has `ID string` and `HasImage bool`

- [ ] **Step 1: Update the apiclient decode test (RED)**

Replace `TestGetPaperParsesCard` in `internal/apiclient/client_test.go` with:

```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/apiclient/ -run TestGetPaperParsesCard -v`
Expected: FAIL — `c.Introduction`/`CardMethodology`/etc. undefined.

- [ ] **Step 3: Update the apiclient structs**

In `internal/apiclient/client.go`, replace the `Card` struct with the v3 types and extend `PaperDetail` + `Figure`:

```go
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
```

Add a `Section` type (next to `Figure`):

```go
type Section struct {
	Order     int32   `json:"order"`
	Heading   *string `json:"heading,omitempty"`
	PageStart *int32  `json:"page_start,omitempty"`
	PageEnd   *int32  `json:"page_end,omitempty"`
}
```

Add `ID` and `HasImage` to `Figure`:

```go
type Figure struct {
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Caption  string `json:"caption"`
	Order    int32  `json:"order"`
	ID       string `json:"id"`
	HasImage bool   `json:"has_image"`
}
```

Add `Abstract` and `Sections` to `PaperDetail` (place after `Title`/`DOI` and before `Authors`):

```go
	Abstract         *string   `json:"abstract,omitempty"`
	Sections         []Section `json:"sections"`
```

- [ ] **Step 4: Run the apiclient test to verify it passes**

Run: `go test ./internal/apiclient/ -v`
Expected: PASS. (The `web` package will not compile yet — fixed in the next steps.)

- [ ] **Step 5: Rewrite the reading template for v3**

Replace the entire contents of `internal/web/templates/paper.tmpl` with:

```
{{define "title"}}{{if .Detail.Title}}{{.Detail.Title}}{{else}}论文{{end}} — ScholarFlow{{end}}
{{define "content"}}
{{$ev := .EvidenceByClaim}}
{{$figs := .FiguresByClaim}}
<p><a href="/">← 返回论文集</a></p>
<h1>{{if .Detail.Title}}{{.Detail.Title}}{{else}}{{.Detail.UploadedFilename}}{{end}}</h1>
<p class="subtitle">
  {{range $i, $a := .Detail.Authors}}{{if $i}}, {{end}}{{$a.DisplayName}}{{end}}
  {{if .Detail.PublicationYear}} · {{.Detail.PublicationYear}}{{end}}
  {{if .Detail.DOI}} · {{.Detail.DOI}}{{end}}
</p>
{{if .Detail.Card}}
  {{$card := .Detail.Card}}
  {{with $card.Introduction}}<section><h2>引言</h2><p>{{.}}{{template "sidenotes" index $ev "introduction"}}{{template "figurenotes" index $figs "introduction"}}</p></section>{{end}}
  {{with $card.RelatedWork}}<section><h2>相关工作</h2><p>{{.}}{{template "sidenotes" index $ev "related_work"}}{{template "figurenotes" index $figs "related_work"}}</p></section>{{end}}
  {{if $card.Methodology}}<section><h2>方法</h2><ul class="card-list">
  {{range $i, $m := $card.Methodology}}<li><span class="meth-problem">{{$m.Problem}}</span> <span class="meth-arrow">→</span> <span class="meth-method">{{$m.Method}}</span>{{$key := printf "methodology#%d" $i}}{{template "sidenotes" index $ev $key}}{{template "figurenotes" index $figs $key}}</li>
  {{end}}</ul></section>{{end}}
  {{if $card.Results}}<section><h2>结果</h2>
  {{range $i, $r := $card.Results}}{{$key := printf "results#%d" $i}}<div class="result">
    <p class="result-metric"><strong>{{$r.Metric}}</strong>{{if $r.SelfOnly}} <span class="badge-self">仅自测</span>{{end}}{{with $r.Finding}}：{{.}}{{end}}{{template "sidenotes" index $ev $key}}{{template "figurenotes" index $figs $key}}</p>
    {{if $r.Comparisons}}<table class="cmp"><thead><tr><th>对比方法</th><th>数值</th><th>引用</th></tr></thead><tbody>
    {{range $c := $r.Comparisons}}<tr><td>{{$c.Work}}</td><td>{{$c.Value}}</td><td>{{$c.Reference}}</td></tr>{{end}}
    </tbody></table>{{end}}
  </div>{{end}}</section>{{end}}
  {{$impl := $card.Implementation}}{{if or $impl.Overview $impl.Modules}}<section><h2>实现</h2>
  {{with $impl.Overview}}<p>{{.}}{{template "sidenotes" index $ev "implementation"}}{{template "figurenotes" index $figs "implementation"}}</p>{{end}}
  {{if $impl.Modules}}<ul class="card-list">
  {{range $i, $mod := $impl.Modules}}{{$key := printf "modules#%d" $i}}<li><strong>{{$mod.Name}}</strong>{{with $mod.Function}} — {{.}}{{end}}{{with $mod.Design}}<br><span class="mod-label">设计：</span>{{.}}{{end}}{{with $mod.Principle}}<br><span class="mod-label">原理：</span>{{.}}{{end}}{{template "sidenotes" index $ev $key}}{{template "figurenotes" index $figs $key}}</li>
  {{end}}</ul>{{end}}
  </section>{{end}}
  {{if or $card.CodeLinks $card.DataLinks}}<section><h2>链接</h2><ul class="card-list">{{range $card.CodeLinks}}<li><a href="{{.}}">{{.}}</a></li>{{end}}{{range $card.DataLinks}}<li><a href="{{.}}">{{.}}</a></li>{{end}}</ul></section>{{end}}
{{else}}
  <p class="notice">阅读尚未完成（状态：{{.Detail.Status}}）。</p>
{{end}}
{{end}}

{{define "sidenotes"}}{{range $i, $e := .}}{{if $i}}<sup class="sidenote-comma">,</sup>{{end}}<label for="{{$e.DOMID}}" class="margin-toggle sidenote-number"></label><input type="checkbox" id="{{$e.DOMID}}" class="margin-toggle"/><span class="sidenote">{{if $e.SectionID}}[§{{$e.SectionID}}] {{end}}{{if $e.Page}}[p.{{$e.Page}}] {{end}}{{$e.Snippet}}</span>{{end}}{{end}}

{{define "figurenotes"}}{{range $f := .}}<label for="{{$f.DOMID}}" class="margin-toggle figure-toggle">&#8853;</label><input type="checkbox" id="{{$f.DOMID}}" class="margin-toggle"/><span class="marginnote figure-callout"><strong>{{$f.Label}}</strong>{{if $f.Page}}（第 {{$f.Page}} 页）{{end}}{{if $f.Caption}}：{{$f.Caption}}{{end}}</span>{{end}}{{end}}
```

- [ ] **Step 6: Add v3 + comment styles to app.css**

Append to `internal/web/static/app.css`:

```css
/* --- v3 reading sections --- */
.meth-arrow { color: #999; }
.meth-method { font-weight: 600; }
.result { margin: 0.6rem 0 1rem; }
.badge-self { font-family: sans-serif; font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.05rem 0.35rem; border: 1px solid #9a6700; color: #9a6700; border-radius: 0.2rem; }
table.cmp { border-collapse: collapse; font-size: 0.9rem; margin-top: 0.3rem; }
table.cmp th, table.cmp td { border-bottom: 1px solid #ddd; padding: 0.2rem 0.6rem 0.2rem 0; text-align: left; }
.mod-label { color: #666; font-size: 0.85rem; }
/* Demand 4: clamp each comment to one line (number + [§]/[p.] lead the line and
   stay visible); expand to full text on hover. */
.sidenote { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sidenote:hover { white-space: normal; overflow: visible; }
```

- [ ] **Step 7: Rewrite the web render tests for v3**

Replace the entire contents of `internal/web/render_test.go` with:

```go
package web

import (
	"strings"
	"testing"

	"scholarflow_web/internal/apiclient"
)

func intPtr(v int) *int { return &v }

func TestBuildPaperViewGroupsByCompositeClaim(t *testing.T) {
	view := BuildPaperView(apiclient.PaperDetail{
		Figures: []apiclient.Figure{{Label: "Figure 2", Caption: "结构图"}},
		Card: &apiclient.Card{
			Evidence: []apiclient.Evidence{
				{ClaimKey: "introduction", SectionID: "3", Snippet: "a"},
				{ClaimKey: "results", ClaimIndex: intPtr(0), SectionID: "4", Snippet: "b"},
				{ClaimKey: "results", ClaimIndex: intPtr(0), SectionID: "5", Snippet: "c"},
			},
			Figures: []apiclient.CardFigure{
				{Label: "figure 2", ClaimKey: "results", ClaimIndex: intPtr(0), Page: intPtr(3)},
			},
		},
	})
	if len(view.EvidenceByClaim["introduction"]) != 1 {
		t.Fatalf("introduction evidence = %#v", view.EvidenceByClaim["introduction"])
	}
	if len(view.EvidenceByClaim["results#0"]) != 2 {
		t.Fatalf("results#0 evidence = %#v", view.EvidenceByClaim["results#0"])
	}
	figs := view.FiguresByClaim["results#0"]
	if len(figs) != 1 || figs[0].Caption != "结构图" || figs[0].Page == nil || *figs[0].Page != 3 {
		t.Fatalf("results#0 figures = %#v", figs)
	}
	seen := map[string]bool{}
	for _, notes := range view.EvidenceByClaim {
		for _, n := range notes {
			if seen[n.DOMID] {
				t.Fatalf("duplicate DOM id %q", n.DOMID)
			}
			seen[n.DOMID] = true
		}
	}
	for _, notes := range view.FiguresByClaim {
		for _, n := range notes {
			if seen[n.DOMID] {
				t.Fatalf("duplicate DOM id %q", n.DOMID)
			}
			seen[n.DOMID] = true
		}
	}
}

func TestRenderCollection(t *testing.T) {
	title := "深度学习论文"
	var b strings.Builder
	err := Render(&b, "collection.tmpl", []apiclient.PaperSummary{
		{PaperID: "p1", Title: &title, Status: "completed", UploadedFilename: "a.pdf"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{"深度学习论文", "/papers/p1", "status-completed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("collection missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderPaperV3Sections(t *testing.T) {
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Card: &apiclient.Card{
			Introduction: "本文研究 X",
			Methodology:  []apiclient.CardMethodology{{Problem: "稀疏", Method: "门控"}},
			Results: []apiclient.CardResult{{
				Metric: "准确率", Finding: "提升4分", SelfOnly: true,
				Comparisons: []apiclient.CardComparison{{Work: "BaseX", Value: "80%", Reference: "[12]"}},
			}},
			Implementation: apiclient.CardImplementation{
				Overview: "整体设计",
				Modules:  []apiclient.CardModule{{Name: "编码器", Function: "编码输入", Principle: "E=mc^2"}},
			},
			Evidence: []apiclient.Evidence{{ClaimKey: "results", ClaimIndex: intPtr(0), SectionID: "3", Page: intPtr(7), Snippet: "证据A"}},
		},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		"本文研究 X", "稀疏", "门控", "准确率", "提升4分", "仅自测",
		"BaseX", "80%", "[12]", "整体设计", "编码器", "E=mc^2",
		// comment keeps both section and page, plus snippet:
		"[§3]", "[p.7]", "证据A", "sidenote",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("paper missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderPaperEmptyCardRendersNoSections(t *testing.T) {
	// A non-nil card with empty v3 fields (e.g. an old 2.0 card decoded into v3)
	// renders the page without erroring and without the no-card notice.
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Title: strPtr("空卡片"),
		Card:  &apiclient.Card{},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "空卡片") {
		t.Fatalf("expected title rendered, got:\n%s", out)
	}
	if strings.Contains(out, "阅读尚未完成") {
		t.Fatalf("non-nil card should not show the no-card notice:\n%s", out)
	}
}

func TestRenderPaperNoCardNotice(t *testing.T) {
	var b strings.Builder
	view := PaperView{Detail: apiclient.PaperDetail{PaperID: "p1", Status: "reading", UploadedFilename: "a.pdf"}}
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(b.String(), "状态：reading") {
		t.Fatalf("expected no-card notice, got:\n%s", b.String())
	}
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 8: Update the handlers test card literal**

In `internal/web/handlers_test.go`, in `TestPaperRendersCard`, replace the `Card` literal and the asserted substrings:

```go
		Card: &apiclient.Card{Introduction: "引言X", Evidence: []apiclient.Evidence{{ClaimKey: "introduction", SectionID: "3", Snippet: "片段"}}},
```
and
```go
	for _, want := range []string{"引言X", "sidenote", "片段"} {
```

- [ ] **Step 9: Run the full suite and build**

Run: `go test ./... 2>&1 | grep -v "no test files"`
Expected: all packages `ok`.

Run: `go build ./...`
Expected: builds with no error.

- [ ] **Step 10: Commit**

```bash
go fmt ./...
git add internal/apiclient/client.go internal/apiclient/client_test.go internal/web/templates/paper.tmpl internal/web/static/app.css internal/web/render_test.go internal/web/handlers_test.go
git commit -m "feat(web): render v3 paper card with truncated hover comments"
```

---

### Task 2: Abstract + thesis outline

**Files:**
- Modify: `internal/web/view.go` (`Outline` on `PaperView`, `OutlineEntry` type)
- Modify: `internal/web/templates/paper.tmpl` (abstract + outline blocks)
- Modify: `internal/web/static/app.css` (block styles)
- Modify: `internal/web/render_test.go` (assert abstract + outline)

**Interfaces:**
- Consumes: `apiclient.PaperDetail.Abstract`, `apiclient.PaperDetail.Sections` (added in Task 1).
- Produces: `PaperView.Outline []OutlineEntry`; `OutlineEntry{Heading string; Page *int}`.

- [ ] **Step 1: Write the failing outline test (RED)**

Add to `internal/web/render_test.go`:

```go
func TestBuildPaperViewOutline(t *testing.T) {
	view := BuildPaperView(apiclient.PaperDetail{
		Sections: []apiclient.Section{
			{Order: 1, Heading: strPtr("Introduction"), PageStart: int32Ptr(1)},
			{Order: 2, Heading: nil},        // skipped (no heading)
			{Order: 3, Heading: strPtr("")}, // skipped (empty heading)
			{Order: 4, Heading: strPtr("Results"), PageStart: int32Ptr(6)},
		},
	})
	if len(view.Outline) != 2 {
		t.Fatalf("outline = %#v, want 2 entries", view.Outline)
	}
	if view.Outline[0].Heading != "Introduction" || view.Outline[0].Page == nil || *view.Outline[0].Page != 1 {
		t.Fatalf("outline[0] = %#v", view.Outline[0])
	}
	if view.Outline[1].Heading != "Results" {
		t.Fatalf("outline[1] = %#v", view.Outline[1])
	}
}

func TestRenderPaperAbstractAndOutline(t *testing.T) {
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Abstract: strPtr("这是原始摘要"),
		Sections: []apiclient.Section{{Order: 1, Heading: strPtr("引言章节"), PageStart: int32Ptr(2)}},
		Card:     &apiclient.Card{Introduction: "正文"},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{"摘要", "这是原始摘要", "目录", "引言章节", "p.2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func int32Ptr(v int32) *int32 { return &v }
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/web/ -run 'Outline|AbstractAndOutline' -v`
Expected: FAIL — `view.Outline` undefined.

- [ ] **Step 3: Add the outline to the view model**

In `internal/web/view.go`, add the type and field, and build it in `BuildPaperView`:

```go
// OutlineEntry is one parsed-section heading for the thesis outline.
type OutlineEntry struct {
	Heading string
	Page    *int
}
```

Add `Outline []OutlineEntry` to the `PaperView` struct. In `BuildPaperView`, before the `if detail.Card == nil` early return, build the outline from the sections (the outline does not depend on the card):

```go
	for _, s := range detail.Sections {
		if s.Heading == nil || strings.TrimSpace(*s.Heading) == "" {
			continue
		}
		view.Outline = append(view.Outline, OutlineEntry{
			Heading: *s.Heading,
			Page:    intFromInt32(s.PageStart),
		})
	}
```

Add the helper at the bottom of `view.go`:

```go
func intFromInt32(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
```

(`strings` is already imported by `view.go`.)

- [ ] **Step 4: Render the abstract + outline blocks**

In `internal/web/templates/paper.tmpl`, insert these two blocks immediately after the closing `</p>` of the `subtitle` paragraph and before `{{if .Detail.Card}}`:

```
{{with .Detail.Abstract}}<section class="abstract-block"><span class="block-label">摘要 / Abstract</span>{{.}}</section>{{end}}
{{if .Outline}}<nav class="outline-block"><span class="block-label">目录 / Outline</span><ol class="outline-list">{{range .Outline}}<li>{{.Heading}}{{if .Page}} <span class="outline-page">p.{{.Page}}</span>{{end}}</li>{{end}}</ol></nav>{{end}}
```

- [ ] **Step 5: Add block styles to app.css**

Append to `internal/web/static/app.css`:

```css
/* --- abstract & outline blocks --- */
.block-label { display: block; font-family: sans-serif; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; color: #666; margin-bottom: 0.2rem; }
.abstract-block { background: #f6f4ec; border-left: 3px solid #b0a98f; padding: 0.6rem 1rem; margin: 0.8rem 0; }
.outline-block { border: 1px dashed #cfc9b6; padding: 0.4rem 1rem; margin: 0.8rem 0; }
.outline-list { margin: 0; }
.outline-page { color: #999; font-size: 0.85rem; }
```

- [ ] **Step 6: Run tests and build**

Run: `go test ./internal/web/ -v`
Expected: PASS (new outline/abstract tests plus the Task 1 tests).

Run: `go build ./...`
Expected: builds with no error.

- [ ] **Step 7: Commit**

```bash
go fmt ./...
git add internal/web/view.go internal/web/templates/paper.tmpl internal/web/static/app.css internal/web/render_test.go
git commit -m "feat(web): add original abstract and thesis outline"
```

---

### Task 3: Figure images (proxy + inline diagram + margin thumbnails)

**Files:**
- Modify: `internal/apiclient/client.go` (`GetFigureImage`)
- Modify: `internal/apiclient/client_test.go` (image test)
- Modify: `internal/web/handlers.go` (`API` interface += `GetFigureImage`; `FigureImage` handler)
- Modify: `internal/web/handlers_test.go` (`fakeAPI.GetFigureImage`; proxy tests)
- Modify: `cmd/web/main.go` (image route + interface)
- Modify: `internal/web/view.go` (`FigureNote` += `HasImage`/`ImageURL`; resolve by label)
- Modify: `internal/web/templates/paper.tmpl` (`figurenotes` thumbnails; `inlinefigures` for implementation)
- Modify: `internal/web/static/app.css` (thumbnail, lightbox, inline figure)
- Modify: `internal/web/render_test.go` (assert `<img>`)

**Interfaces:**
- Consumes: `apiclient.Figure.ID`/`HasImage`, `PaperView.FiguresByClaim` (Task 1).
- Produces:
  - `apiclient.Client.GetFigureImage(ctx context.Context, paperID, figureID string) (io.ReadCloser, string, error)`
  - `web.Handler.FigureImage(http.ResponseWriter, *http.Request)` on route `GET /papers/{id}/figures/{figureId}/image`
  - `FigureNote` gains `HasImage bool` and `ImageURL string`.

- [ ] **Step 1: Write the failing apiclient image test (RED)**

Add to `internal/apiclient/client_test.go`:

```go
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
```

Add `"io"` to the `client_test.go` imports.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/apiclient/ -run TestGetFigureImage -v`
Expected: FAIL — `GetFigureImage` undefined.

- [ ] **Step 3: Implement `GetFigureImage`**

In `internal/apiclient/client.go`, add (the file already imports `context`, `fmt`, `io`, `net/http`):

```go
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
```

- [ ] **Step 4: Run the apiclient test to verify it passes**

Run: `go test ./internal/apiclient/ -run TestGetFigureImage -v`
Expected: PASS.

- [ ] **Step 5: Write the failing handler proxy test (RED)**

In `internal/web/handlers_test.go`, add a `GetFigureImage` method to `fakeAPI` and proxy tests. First add the fields + method (place after the existing `GetPaper` method):

```go
// add to fakeAPI struct:
//   imgBody     string
//   imgType     string
//   imgErr      error

func (f *fakeAPI) GetFigureImage(ctx context.Context, paperID, figureID string) (io.ReadCloser, string, error) {
	if f.imgErr != nil {
		return nil, "", f.imgErr
	}
	return io.NopCloser(strings.NewReader(f.imgBody)), f.imgType, nil
}
```

Add `"io"` to the `handlers_test.go` imports. Extend `routerFor` to register the image route:

```go
	r.Get("/papers/{id}/figures/{figureId}/image", h.FigureImage)
```

Add the tests:

```go
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
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/web/ -run FigureImage -v`
Expected: FAIL — `FigureImage` undefined; `API` interface not satisfied by `fakeAPI` yet only if interface updated. (Build error is the expected RED.)

- [ ] **Step 7: Implement the handler and extend the interface**

In `internal/web/handlers.go`, add `"errors"` and `"io"` to imports if missing (`errors` is already imported; add `io`). Extend the `API` interface:

```go
type API interface {
	ListPapers(ctx context.Context) ([]apiclient.PaperSummary, error)
	GetPaper(ctx context.Context, id string) (apiclient.PaperDetail, error)
	GetFigureImage(ctx context.Context, paperID, figureID string) (io.ReadCloser, string, error)
}
```

Add the handler:

```go
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
```

- [ ] **Step 8: Register the route in main**

In `cmd/web/main.go`, add the route after the existing `/papers/{id}` route:

```go
	r.Get("/papers/{id}/figures/{figureId}/image", h.FigureImage)
```

- [ ] **Step 9: Resolve figure image URLs in the view**

In `internal/web/view.go`, add `HasImage bool` and `ImageURL string` to the `FigureNote` struct. In `BuildPaperView`, build a label→figure map carrying ID/HasImage and set the URL. Replace the existing `captionByLabel` construction and the figure loop with:

```go
	figByLabel := make(map[string]apiclient.Figure, len(detail.Figures))
	for _, f := range detail.Figures {
		figByLabel[normalizeLabel(f.Label)] = f
	}
	view.FiguresByClaim = make(map[string][]FigureNote)
	for _, f := range detail.Card.Figures {
		id++
		key := claimKey(f.ClaimKey, f.ClaimIndex)
		src := figByLabel[normalizeLabel(f.Label)]
		note := FigureNote{
			DOMID:    fmt.Sprintf("fn-%d", id),
			Label:    f.Label,
			Page:     f.Page,
			Caption:  src.Caption,
			HasImage: src.HasImage,
		}
		if src.HasImage {
			note.ImageURL = "/papers/" + detail.PaperID + "/figures/" + src.ID + "/image"
		}
		view.FiguresByClaim[key] = append(view.FiguresByClaim[key], note)
	}
```

- [ ] **Step 10: Render images in the template**

In `internal/web/templates/paper.tmpl`, replace the `figurenotes` sub-template and add an `inlinefigures` sub-template:

```
{{define "figurenotes"}}{{range $f := .}}<label for="{{$f.DOMID}}" class="margin-toggle figure-toggle">&#8853;</label><input type="checkbox" id="{{$f.DOMID}}" class="margin-toggle"/><span class="marginnote figure-callout"><strong>{{$f.Label}}</strong>{{if $f.Page}}（第 {{$f.Page}} 页）{{end}}{{if $f.Caption}}：{{$f.Caption}}{{end}}{{if $f.HasImage}}<label class="thumb-toggle"><img class="margin-thumb" src="{{$f.ImageURL}}" alt="{{$f.Label}}"><input type="checkbox" class="lightbox-toggle"><span class="lightbox"><img src="{{$f.ImageURL}}" alt="{{$f.Label}}"></span></label>{{end}}</span>{{end}}{{end}}

{{define "inlinefigures"}}{{range $f := .}}{{if $f.HasImage}}<figure class="inline-figure"><img src="{{$f.ImageURL}}" alt="{{$f.Label}}"><figcaption>{{$f.Label}}{{if $f.Page}}（第 {{$f.Page}} 页）{{end}}{{if $f.Caption}}：{{$f.Caption}}{{end}}</figcaption></figure>{{else}}<label for="{{$f.DOMID}}" class="margin-toggle figure-toggle">&#8853;</label><input type="checkbox" id="{{$f.DOMID}}" class="margin-toggle"/><span class="marginnote figure-callout"><strong>{{$f.Label}}</strong>{{if $f.Page}}（第 {{$f.Page}} 页）{{end}}{{if $f.Caption}}：{{$f.Caption}}{{end}}</span>{{end}}{{end}}{{end}}
```

In the implementation section, switch the overview's figure rendering to inline. Change the `{{with $impl.Overview}}` line to use `inlinefigures`:

```
  {{with $impl.Overview}}<p>{{.}}{{template "sidenotes" index $ev "implementation"}}{{template "inlinefigures" index $figs "implementation"}}</p>{{end}}
```

- [ ] **Step 11: Add figure-image styles to app.css**

Append to `internal/web/static/app.css`:

```css
/* --- figure images --- */
.inline-figure { margin: 0.8rem 0; }
.inline-figure img { max-width: 100%; border: 1px solid #ddd; }
.inline-figure figcaption { font-size: 0.85rem; color: #777; margin-top: 0.3rem; }
.margin-thumb { display: block; max-width: 100%; margin-top: 0.3rem; border: 1px solid #ddd; cursor: zoom-in; }
.lightbox-toggle { display: none; }
.lightbox { display: none; }
.lightbox-toggle:checked ~ .lightbox { display: flex; position: fixed; inset: 0; background: rgba(0,0,0,0.8); align-items: center; justify-content: center; z-index: 50; cursor: zoom-out; }
.lightbox img { max-width: 92%; max-height: 92%; border: none; }
```

- [ ] **Step 12: Assert an image renders (RED→GREEN)**

Add to `internal/web/render_test.go`:

```go
func TestRenderPaperFigureImages(t *testing.T) {
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Figures: []apiclient.Figure{
			{Label: "Figure 1", Caption: "架构图", ID: "f1", HasImage: true},
			{Label: "Figure 3", Caption: "曲线", ID: "f3", HasImage: true},
		},
		Card: &apiclient.Card{
			Implementation: apiclient.CardImplementation{Overview: "系统总体"},
			Results:        []apiclient.CardResult{{Metric: "acc", Finding: "好"}},
			Figures: []apiclient.CardFigure{
				{Label: "Figure 1", ClaimKey: "implementation"},
				{Label: "Figure 3", ClaimKey: "results", ClaimIndex: intPtr(0)},
			},
		},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	// Architecture figure renders inline; chart figure renders as a margin thumbnail.
	if !strings.Contains(out, `class="inline-figure"`) || !strings.Contains(out, "/papers/p1/figures/f1/image") {
		t.Fatalf("architecture inline figure missing:\n%s", out)
	}
	if !strings.Contains(out, "margin-thumb") || !strings.Contains(out, "/papers/p1/figures/f3/image") {
		t.Fatalf("chart margin thumbnail missing:\n%s", out)
	}
}
```

Run: `go test ./internal/web/ -run FigureImages -v`
Expected: after Steps 9-11 it PASSES (run before Step 9 to see RED if desired).

- [ ] **Step 13: Run the full suite and build**

Run: `go test ./... 2>&1 | grep -v "no test files"`
Expected: all packages `ok`.

Run: `go build ./...`
Expected: builds with no error.

- [ ] **Step 14: Commit**

```bash
go fmt ./...
git add internal/apiclient/client.go internal/apiclient/client_test.go internal/web/handlers.go internal/web/handlers_test.go cmd/web/main.go internal/web/view.go internal/web/templates/paper.tmpl internal/web/static/app.css internal/web/render_test.go
git commit -m "feat(web): render extracted figure images via image proxy"
```

---

## Self-Review

**Spec coverage:**
- v3 card contract in apiclient → Task 1 Step 3. ✓
- v3 reading template (intro/related_work/methodology pairs/results+comparisons+self_only/implementation+modules/links) → Task 1 Step 5. ✓
- Demand 4 (truncate + hover, number/§/page kept) → Task 1 Steps 5 (template shows both §+page) + 6 (clamp/hover CSS). ✓
- Demand 2 abstract → Task 2. ✓
- Demand 1 outline (from parsed sections, headings+pages) → Task 2. ✓
- Demand 3 figures: image proxy (route+handler+apiclient method), inline architecture diagram, margin thumbnails + lightbox, text fallback → Task 3. ✓
- No server change, web-only, no JS, custom CSS in app.css → honored across tasks. ✓
- Tests updated (client_test, render_test, handlers_test) + new view/handler/image tests → all tasks. ✓

**Placeholder scan:** No TBD/TODO; every code/template/CSS step shows full content.

**Type consistency:** `CardMethodology/CardResult/CardComparison/CardModule/CardImplementation` field names + JSON tags (Task 1 Step 3) match the template field access (`$m.Problem`, `$r.Metric`, `$c.Work`, `$mod.Name`, `$impl.Overview`) and the test literals. `FigureNote.HasImage/ImageURL` (Task 3 Step 9) match the template (`$f.HasImage`, `$f.ImageURL`) and the render test. `OutlineEntry{Heading, Page}` (Task 2) matches the template (`.Heading`, `.Page`) and tests. `GetFigureImage(ctx, paperID, figureID) (io.ReadCloser, string, error)` is identical across apiclient impl, the `API` interface, the `fakeAPI`, and the handler call. The `strPtr`/`int32Ptr`/`intPtr` test helpers are each defined once in `render_test.go`.
