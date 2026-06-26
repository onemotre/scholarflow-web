# Editorial Viewer Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Tufte-based reading layout with a custom "Editorial" layout that fixes column alignment and image overflow, supports light/dark via the OS, and keeps the two-column content + evidence presentation.

**Architecture:** Server-rendered Go `html/template` (unchanged). Each evidence-bearing unit ("claim") renders as a CSS-Grid row: body cell + aligned evidence/figure cell. `tufte.css` is removed; a new `app.css` carries the whole theme. Theme follows `prefers-color-scheme` (no JS). The view-model (`web.BuildPaperView`) is reused; only the now-unused checkbox `DOMID` fields are dropped.

**Tech Stack:** Go `html/template` (embedded), plain CSS with custom properties, `chi` router (unchanged).

## Global Constraints

- **No JavaScript.** All interactivity/theming is CSS-only; theme = `prefers-color-scheme`.
- **Module:** `scholarflow-web` (run all commands from `/home/onemotre/workspace/scholarflow/scholarflow-web`).
- **Images must never overflow their column:** every `img` is `max-width:100%`.
- **Keep existing view-model class names** where they already exist (`.abstract-block`, `.outline-*`, `.meth-*`, `.result-metric`, `.badge-self`, `.cmp`, `.card-list`, `.subtitle`, `.paper-list`, `.status-badge`, `.notice`, `.block-label`, `.paper-meta`). New classes: `.paper`, `.claim`, `.claim-body`, `.claim-notes`, `.note`, `.marker`, `.snip`, `.figure-note`, `.figure-inline`.
- Verify with `go build ./...` and `go test ./...`; tests must stay green.

---

## File Structure

- `internal/web/templates/base.tmpl` — modify: drop `tufte.css` link, wrap content in `<article class="paper">`.
- `internal/web/static/app.css` — replace entirely with the Editorial stylesheet (theme tokens, base/collection/error styles, claim grid, figures).
- `internal/web/static/tufte.css` — delete.
- `internal/web/templates/paper.tmpl` — rewrite the card body to claim-grid rows + a `claimnotes` sub-template; drop the `sidenotes`/`figurenotes` checkbox-hack sub-templates; keep `inlinefigures` for the implementation architecture figure.
- `internal/web/view.go` — remove the now-unused `DOMID` fields and the `id` counter (no checkboxes anymore).
- `internal/web/render_test.go` — update assertions to the new markup.

---

### Task 1: Editorial stylesheet + base shell

**Files:**
- Modify: `internal/web/templates/base.tmpl`
- Create/replace: `internal/web/static/app.css`
- Delete: `internal/web/static/tufte.css`
- Test: `internal/web/render_test.go`

**Interfaces:**
- Produces: a `.paper` page container and all theme tokens/classes consumed by `paper.tmpl` in Task 2. Class contract listed in Global Constraints.

- [ ] **Step 1: Write a failing test that base no longer loads tufte.css**

Add to `internal/web/render_test.go`:

```go
func TestBaseUsesEditorialStylesheetOnly(t *testing.T) {
	var b strings.Builder
	if err := Render(&b, "collection.tmpl", []apiclient.PaperSummary(nil)); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if strings.Contains(out, "tufte.css") {
		t.Fatalf("base still links tufte.css:\n%s", out)
	}
	if !strings.Contains(out, "/static/app.css") {
		t.Fatalf("base does not link app.css:\n%s", out)
	}
	if !strings.Contains(out, `class="paper"`) {
		t.Fatalf("base does not wrap content in .paper:\n%s", out)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/web/ -run TestBaseUsesEditorialStylesheetOnly -v`
Expected: FAIL (base still has `tufte.css`, no `.paper`).

- [ ] **Step 3: Update `base.tmpl`**

Replace the whole file with:

```html
{{define "base"}}<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>{{block "title" .}}ScholarFlow{{end}}</title>
<link rel="stylesheet" href="/static/app.css">
</head>
<body>
<article class="paper">
{{block "content" .}}{{end}}
</article>
</body>
</html>{{end}}
```

- [ ] **Step 4: Replace `internal/web/static/app.css` with the Editorial stylesheet**

```css
/* ScholarFlow viewer — "Editorial" layout. Theme follows the OS via
   prefers-color-scheme (no JS). Two-column claims use CSS grid so evidence
   aligns beside its claim and images never overflow. */
:root{
  --bg:#fcfcfd; --surface:#f4f4f8; --fg:#23232b; --muted:#6b6b78;
  --rule:#e3e3ec; --accent:#5b4bd6; --accent-ink:#4636c0;
  --hl-bg:rgba(91,75,214,.12); --hl-line:#5b4bd6;
  --ok:#1a7f37; --warn:#9a6700; --err:#b42318;
  --note-w:18rem; --gap:2.4rem; --content:42rem;
}
@media (prefers-color-scheme:dark){
  :root{ --bg:#15151b; --surface:#1e1e27; --fg:#e8e8f0; --muted:#a0a0b0;
    --rule:#2c2c39; --accent:#a99bff; --accent-ink:#c3b8ff;
    --hl-bg:rgba(169,155,255,.20); --hl-line:#a99bff;
    --ok:#4ac776; --warn:#d8a93a; --err:#ff7b6b; }
}
*{box-sizing:border-box}
html{color-scheme:light dark}
body{margin:0;background:var(--bg);color:var(--fg);
  font-family:-apple-system,system-ui,"Segoe UI","Noto Sans CJK SC","Source Han Sans SC",sans-serif;
  font-size:16px;line-height:1.7}
img{max-width:100%;height:auto;display:block}
a{color:var(--accent-ink)}
h1,h2,.paper-title{font-family:Georgia,"Songti SC","Noto Serif CJK SC",serif}

.paper{max-width:calc(var(--content) + var(--note-w) + var(--gap));margin:0 auto;padding:2rem 1.5rem 6rem}
h1{font-size:2rem;line-height:1.2;margin:.4rem 0 .2rem;letter-spacing:-.01em}
.subtitle{color:var(--muted);margin:.3rem 0 1.4rem}
.block-label{display:block;font-size:.68rem;letter-spacing:.12em;text-transform:uppercase;color:var(--accent-ink);margin-bottom:.35rem}

/* collection + error pages */
.paper-list{list-style:none;padding:0}
.paper-list li{margin:.9rem 0}
.paper-list a{font-weight:600;text-decoration:none}
.paper-meta{color:var(--muted);font-size:.85rem;margin-top:.15rem}
.status-badge{font-size:.66rem;text-transform:uppercase;letter-spacing:.05em;padding:.08rem .45rem;border:1px solid currentColor;border-radius:.3rem;margin-left:.4rem}
.status-completed{color:var(--ok)} .status-failed{color:var(--err)}
.status-reading,.status-processing,.status-parsed,.status-queued{color:var(--warn)}
.notice{background:var(--surface);border-left:3px solid var(--warn);padding:.6rem 1rem;border-radius:.3rem}

/* abstract + outline */
.abstract-block{border-top:2px solid var(--accent);padding:.9rem 0 0;margin:0 0 1.6rem}
.outline-block{margin:0 0 2rem}
.outline-list{list-style:none;margin:0;padding:0;columns:2;column-gap:2rem}
.outline-item{padding:.12rem 0;break-inside:avoid}
.outline-num{color:var(--accent-ink);font-variant-numeric:tabular-nums;margin-right:.35rem}
.outline-page{color:var(--muted);font-size:.8rem}
.outline-l0{font-weight:700}
.outline-l1{margin-left:1.2rem} .outline-l2{margin-left:2.4rem} .outline-l3{margin-left:3.6rem}
.outline-l4{margin-left:4.8rem} .outline-l5{margin-left:6rem} .outline-l6{margin-left:7.2rem}

h2{font-size:1.4rem;margin:2.4rem 0 .8rem}

/* claim grid: body | evidence notes, aligned, never overflowing */
.claim{display:grid;grid-template-columns:minmax(0,1fr) var(--note-w);gap:var(--gap);align-items:start;margin:1.2rem 0}
.claim-body{min-width:0}
.claim-body p{margin:.3rem 0}
.claim-notes{min-width:0;border-left:2px solid var(--rule);padding-left:1.2rem}
.note{font-size:.84rem;line-height:1.55;color:var(--muted);margin-bottom:.8rem}
.marker{display:inline-block;font-size:.64rem;font-weight:700;letter-spacing:.03em;padding:.05rem .4rem;margin-right:.25rem;border-radius:.25rem;color:var(--accent-ink);border:1px solid var(--accent)}
.snip{color:var(--fg)}
.figure-note{margin:.5rem 0 .9rem}
.figure-note img{border-radius:.5rem;box-shadow:0 1px 4px rgba(0,0,0,.18);max-height:14rem;width:auto}
.figure-note figcaption{font-size:.76rem;color:var(--muted);margin-top:.3rem}
.figure-inline{margin:1rem 0}
.figure-inline img{border-radius:.6rem;box-shadow:0 2px 10px rgba(0,0,0,.2)}
.figure-inline figcaption{font-size:.84rem;color:var(--muted);margin-top:.5rem;line-height:1.55;border-left:2px solid var(--accent);padding-left:.7rem}

/* card pieces */
.cmp{border-collapse:collapse;font-size:.9rem;margin:.5rem 0}
.cmp th{color:var(--muted);font-weight:600}
.cmp th,.cmp td{border-bottom:1px solid var(--rule);padding:.3rem .8rem .3rem 0;text-align:left}
.meth-problem{color:var(--muted)} .meth-arrow{color:var(--accent)} .meth-method{font-weight:600}
.mod-label{color:var(--muted);font-size:.85rem}
.result-metric{margin:.2rem 0}
.badge-self{font-size:.62rem;text-transform:uppercase;letter-spacing:.05em;padding:.05rem .4rem;border-radius:.25rem;border:1px solid var(--accent);color:var(--accent-ink)}
.card-list{list-style:none;padding:0;margin:.3rem 0}

@media (max-width:840px){
  .outline-list{columns:1}
  .claim{grid-template-columns:1fr;gap:.7rem}
  .claim-notes{border-left:0;border-top:2px solid var(--rule);padding:.6rem 0 0}
  .figure-note img{max-height:none}
}
```

- [ ] **Step 5: Delete tufte.css**

Run: `git rm internal/web/static/tufte.css`
Expected: file removed (it is no longer referenced).

- [ ] **Step 6: Run the test + build**

Run: `go test ./internal/web/ -run TestBaseUsesEditorialStylesheetOnly -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 7: Commit**

```bash
git add internal/web/templates/base.tmpl internal/web/static/app.css internal/web/render_test.go
git rm internal/web/static/tufte.css
git commit -m "feat(web): editorial stylesheet + base shell, drop tufte.css"
```

---

### Task 2: Claim-grid paper template

**Files:**
- Modify: `internal/web/templates/paper.tmpl`
- Modify: `internal/web/view.go`
- Test: `internal/web/render_test.go`

**Interfaces:**
- Consumes: `PaperView` (`Outline`, `EvidenceByClaim`, `FiguresByClaim`) and the `dict` funcMap helper (already in `render.go`).
- Produces: each claim as `<div class="claim"><div class="claim-body">…</div><aside class="claim-notes">…</aside></div>`; evidence as `<div class="note"><span class="marker">§N</span>…<span class="snip">…</span></div>`; the implementation architecture figure inline via `inlinefigures`.

- [ ] **Step 1: Remove the unused checkbox `DOMID` fields from `view.go`**

In `internal/web/view.go`, delete `DOMID` from both structs and the `id` counter that fed them:

```go
type EvidenceNote struct {
	Page      *int
	SectionID string
	Snippet   string
}

type FigureNote struct {
	Label    string
	Page     *int
	Caption  string
	HasImage bool
	ImageURL string
}
```

In `BuildPaperView`, delete the `id := 0` line and the `id++` lines, and drop `DOMID: fmt.Sprintf(...)` from both note literals. If `fmt` becomes unused, remove its import (it is still used by `claimKey`, so keep it).

- [ ] **Step 2: Update render tests to the new markup (failing)**

Replace the body-asserting tests `TestRenderPaperV3Sections` and `TestRenderPaperFigureImages` bodies (keep their `BuildPaperView` setup) so they assert the new structure. Example for the sections test — assert the claim grid + markers instead of sidenote markup:

```go
func TestRenderPaperV3Sections(t *testing.T) {
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Card: &apiclient.Card{
			Introduction: "引言正文",
			Methodology:  []apiclient.CardMethodology{{Problem: "难题", Method: "办法"}},
			Results: []apiclient.CardResult{{Metric: "acc", Finding: "好",
				Comparisons: []apiclient.CardComparison{{Work: "BL", Value: "9", Reference: "[1]"}}}},
			Implementation: apiclient.CardImplementation{Overview: "总体",
				Modules: []apiclient.CardModule{{Name: "M1", Function: "f", Design: "d", Principle: "p"}}},
			Evidence: []apiclient.Evidence{{ClaimKey: "results", ClaimIndex: intPtr(0),
				SectionID: "3", Page: intPtr(7), Snippet: "证据片段"}},
		},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`class="claim"`, `class="claim-body"`, `class="claim-notes"`,
		"引言正文", "难题", "办法", "acc", "BL",
		`class="marker"`, "§3", "p.7", "证据片段", "总体", "M1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "margin-toggle") || strings.Contains(out, "sidenote") {
		t.Fatalf("old Tufte markup still present:\n%s", out)
	}
}
```

For `TestRenderPaperFigureImages`, keep the setup; change the assertions to: `class="figure-inline"` present (implementation figure), `class="figure-note"` present (results figure), and `/papers/p1/figures/f3/image` present.

- [ ] **Step 3: Run the tests to verify they fail**

Run: `go test ./internal/web/ -run 'TestRenderPaperV3Sections|TestRenderPaperFigureImages' -v`
Expected: FAIL (template still emits old sidenote markup).

- [ ] **Step 4: Rewrite `internal/web/templates/paper.tmpl`**

```html
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
{{with .Detail.Abstract}}<section class="abstract-block"><span class="block-label">摘要 / Abstract</span>{{.}}</section>{{end}}
{{if .Outline}}<nav class="outline-block"><span class="block-label">目录 / Outline</span><ul class="outline-list">{{range .Outline}}<li class="outline-item outline-l{{.Level}}">{{if .Number}}<span class="outline-num">{{.Number}}</span> {{end}}{{.Heading}}{{if .Page}} <span class="outline-page">p.{{.Page}}</span>{{end}}</li>{{end}}</ul></nav>{{end}}
{{if .Detail.Card}}
  {{$card := .Detail.Card}}
  {{with $card.Introduction}}<section><h2>引言</h2><div class="claim"><div class="claim-body"><p>{{.}}</p></div>{{template "claimnotes" (dict "ev" (index $ev "introduction") "figs" (index $figs "introduction"))}}</div></section>{{end}}
  {{with $card.RelatedWork}}<section><h2>相关工作</h2><div class="claim"><div class="claim-body"><p>{{.}}</p></div>{{template "claimnotes" (dict "ev" (index $ev "related_work") "figs" (index $figs "related_work"))}}</div></section>{{end}}
  {{if $card.Methodology}}<section><h2>方法</h2>
  {{range $i, $m := $card.Methodology}}{{$key := printf "methodology#%d" $i}}<div class="claim"><div class="claim-body"><p><span class="meth-problem">{{$m.Problem}}</span> <span class="meth-arrow">→</span> <span class="meth-method">{{$m.Method}}</span></p></div>{{template "claimnotes" (dict "ev" (index $ev $key) "figs" (index $figs $key))}}</div>
  {{end}}</section>{{end}}
  {{if $card.Results}}<section><h2>结果</h2>
  {{range $i, $r := $card.Results}}{{$key := printf "results#%d" $i}}<div class="claim"><div class="claim-body">
    <p class="result-metric"><strong>{{$r.Metric}}</strong>{{if $r.SelfOnly}} <span class="badge-self">仅自测</span>{{end}}{{with $r.Finding}}：{{.}}{{end}}</p>
    {{if $r.Comparisons}}<table class="cmp"><thead><tr><th>对比方法</th><th>数值</th><th>引用</th></tr></thead><tbody>
    {{range $c := $r.Comparisons}}<tr><td>{{$c.Work}}</td><td>{{$c.Value}}</td><td>{{$c.Reference}}</td></tr>{{end}}
    </tbody></table>{{end}}
  </div>{{template "claimnotes" (dict "ev" (index $ev $key) "figs" (index $figs $key))}}</div>{{end}}</section>{{end}}
  {{$impl := $card.Implementation}}{{if or $impl.Overview $impl.Modules}}<section><h2>实现</h2>
  {{if $impl.Overview}}<div class="claim"><div class="claim-body"><p>{{$impl.Overview}}</p>{{template "inlinefigures" index $figs "implementation"}}</div>{{template "claimnotes" (dict "ev" (index $ev "implementation") "figs" nil)}}</div>{{end}}
  {{range $i, $mod := $impl.Modules}}{{$key := printf "modules#%d" $i}}<div class="claim"><div class="claim-body"><p><strong>{{$mod.Name}}</strong>{{with $mod.Function}} — {{.}}{{end}}{{with $mod.Design}}<br><span class="mod-label">设计：</span>{{.}}{{end}}{{with $mod.Principle}}<br><span class="mod-label">原理：</span>{{.}}{{end}}</p></div>{{template "claimnotes" (dict "ev" (index $ev $key) "figs" (index $figs $key))}}</div>
  {{end}}</section>{{end}}
  {{if or $card.CodeLinks $card.DataLinks}}<section><h2>链接</h2><ul class="card-list">{{range $card.CodeLinks}}<li><a href="{{.}}">{{.}}</a></li>{{end}}{{range $card.DataLinks}}<li><a href="{{.}}">{{.}}</a></li>{{end}}</ul></section>{{end}}
{{else}}
  <p class="notice">阅读尚未完成（状态：{{.Detail.Status}}）。</p>
{{end}}
{{end}}

{{define "claimnotes"}}{{if or .ev .figs}}<aside class="claim-notes">{{range .ev}}<div class="note">{{if .SectionID}}<span class="marker">§{{.SectionID}}</span>{{end}}{{if .Page}}<span class="marker">p.{{.Page}}</span>{{end}}<span class="snip">{{.Snippet}}</span></div>{{end}}{{range .figs}}{{if .HasImage}}<figure class="figure-note"><img src="{{.ImageURL}}" alt="{{.Label}}"><figcaption><strong>{{.Label}}</strong>{{if .Page}}（第 {{.Page}} 页）{{end}}{{if .Caption}}：{{.Caption}}{{end}}</figcaption></figure>{{else}}<div class="note"><span class="marker">{{.Label}}</span>{{if .Page}}<span class="marker">p.{{.Page}}</span>{{end}}{{if .Caption}}<span class="snip">{{.Caption}}</span>{{end}}</div>{{end}}{{end}}</aside>{{end}}{{end}}

{{define "inlinefigures"}}{{range $f := .}}{{if $f.HasImage}}<figure class="figure-inline"><img src="{{$f.ImageURL}}" alt="{{$f.Label}}"><figcaption><strong>{{$f.Label}}</strong>{{if $f.Page}}（第 {{$f.Page}} 页）{{end}}{{if $f.Caption}}：{{$f.Caption}}{{end}}</figcaption></figure>{{else}}<div class="note"><span class="marker">{{$f.Label}}</span>{{if $f.Page}}<span class="marker">p.{{$f.Page}}</span>{{end}}{{if $f.Caption}}<span class="snip">{{$f.Caption}}</span>{{end}}</div>{{end}}{{end}}{{end}}
```

- [ ] **Step 5: Run the render tests**

Run: `go test ./internal/web/ -run 'TestRenderPaperV3Sections|TestRenderPaperFigureImages' -v`
Expected: PASS.

- [ ] **Step 6: Run the full web suite + build**

Run: `go test ./... && go build ./...`
Expected: all PASS, clean build. (If other tests referenced `DOMID`, update them to drop that field.)

- [ ] **Step 7: Commit**

```bash
git add internal/web/templates/paper.tmpl internal/web/view.go internal/web/render_test.go
git commit -m "feat(web): claim-grid paper layout with aligned evidence column"
```

---

### Task 3: Visual verification & polish

**Files:**
- Modify (only if a defect is found): `internal/web/static/app.css`, `internal/web/templates/paper.tmpl`

**Interfaces:**
- Consumes: the running API at `http://localhost:8080` and the SONIC paper `b25ba0e1-08dc-48b0-b233-f5f11a2bd498` (status `parsed`, has card + figures).

- [ ] **Step 1: Build and launch a local web binary**

```bash
go build -o /tmp/webtest ./cmd/web
SCHOLARFLOW_API_URL=http://localhost:8080 WEB_ADDR=:8099 /tmp/webtest &
```
Then confirm: `curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8099/healthz` → `200`.

- [ ] **Step 2: Capture the paper page in light and dark, desktop and narrow**

Using the Playwright MCP browser tools, navigate to
`http://localhost:8099/papers/b25ba0e1-08dc-48b0-b233-f5f11a2bd498` and screenshot:
1. desktop (width 1200) — light, then with `prefers-color-scheme: dark` emulated.
2. narrow (width 480) — confirm the claim collapses to one column.

- [ ] **Step 3: Verify the acceptance criteria against the screenshots**

Confirm, via measurement (`getBoundingClientRect`) and the images:
- No `.claim img` / `.figure-inline img` extends past its container's right edge.
- Every `.claim-body` shares the same left x; every `.claim-notes` shares the same left x (columns aligned).
- The implementation architecture figure renders inline with its caption directly below.
- Markers and links are legible in dark mode (accent uses the brightened dark tokens).

- [ ] **Step 4: Fix any defect found, then re-verify**

If a check fails, adjust `app.css` (or `paper.tmpl`) minimally, rebuild the local binary, and repeat Steps 2–3. Re-run `go test ./...` after any change.

- [ ] **Step 5: Stop the local binary and commit any fixes**

```bash
pkill -f /tmp/webtest
# only if Step 4 changed files:
git add internal/web/static/app.css internal/web/templates/paper.tmpl
git commit -m "fix(web): editorial layout visual polish"
```

---

## Notes for the implementer

- The `dict` funcMap helper already exists in `internal/web/render.go`; `claimnotes` relies on it.
- `claimnotes` is given `"figs" nil` for the implementation overview claim on purpose — implementation-anchored figures render inline (architecture diagram), not in the notes column.
- Do not edit any server module; this is `scholarflow-web` only.
- The CHANGELOG entry and merge are handled after all tasks pass (not part of these tasks).
