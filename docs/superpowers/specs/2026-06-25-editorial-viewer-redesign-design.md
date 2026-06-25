# Editorial Viewer Redesign — Design Spec

Date: 2026-06-25
Module: `scholarflow-web`
Status: approved (design selected from mockups)

## Goal

Replace the Tufte-based reading layout with a custom **"Editorial"** layout (mockup
"Design C") that fixes the recurring layout bugs and meets the user's requirements:

1. **Bilingual CN/EN** body and labels.
2. **Easy image + caption insertion** that never overflows its column.
3. Keep the **two-column content + evidence ("Comment")** layout, but with columns
   that align reliably.
4. Works in **light and dark** themes.
5. **Intuitive highlight colors** that stay visible in dark mode.

The root cause of the prior bugs was building the two-column / margin-note layout on
Tufte's float + negative-margin mechanics. The redesign uses **CSS Grid per claim**
so each claim's body and its evidence sit in aligned grid cells — no floats, no
overflow.

## Non-goals

- No change to the server API or data model.
- No JavaScript. Theme follows the OS via `prefers-color-scheme` (decided: auto-follow,
  no in-page toggle).
- No change to the figure image-proxy route or the card schema.

## Architecture (unchanged pieces)

Server-rendered Go `html/template`, embedded, served by `cmd/web`. The view-model
`web.BuildPaperView` already produces everything the new layout needs:

- `Outline []OutlineEntry` — `Number`, `Heading`, `Page`, `Level`.
- `EvidenceByClaim map[string][]EvidenceNote` — keyed by `claimKey(field, index)`.
- `FiguresByClaim map[string][]FigureNote` — same keying, with `HasImage`/`ImageURL`/`Caption`.

So `view.go` needs little or no change. The work is in the **templates** and **CSS**.

## Layout structure (Design C)

Centered single page, max-width = content + note column + gap. Top to bottom:

- **Header**: `.paper-title` (serif), `.paper-meta` (authors · year · DOI).
- **Abstract**: `.abstract` with a top accent border + `摘要 / Abstract` label.
- **Outline**: `.outline` — a list rendered in **2 CSS columns**, each item showing the
  real section number, heading, and `p.N`, indented by `Level` (`.lvl-0` bold, `.lvl-1`
  indented, …). No `<ol>` auto-numbering.
- **Sections** (`引言`, `相关工作`, `方法`, `结果`, `实现`, `链接`), serif `<h2>`.

### The claim row (the core fix)

Every unit that can carry evidence/figures is a **claim**: an intro/related paragraph,
a methodology pair, a result, an implementation-overview paragraph, or a module. Each
renders as:

```
<div class="claim">
  <div class="claim-body"> …content… </div>
  <aside class="claim-notes"> …evidence + figure notes for this claim… </aside>
</div>
```

`.claim` is `display:grid; grid-template-columns: minmax(0,1fr) var(--note-w);
gap:var(--gap); align-items:start`. The `minmax(0,1fr)` lets the body shrink so nothing
overflows; the notes column is a fixed width. A divider (`border-left`) separates the
notes. Below `--bp` (≈840px) it collapses to one column with the notes under the body.

- `.claim-notes` holds `.note` (evidence) blocks and `.figure-note` figures.
- `.note`: `.marker` chips (`§N`, `p.N`) + `.snip` text with `.hl` highlight spans.
- `.figure-note img`: `max-width:100%; max-height:~14rem; width:auto` (bounded; tall
  figures get a taller cap). `figcaption` below.
- The **architecture figure** (card figure anchored to `implementation`) renders inline
  in the body column as a section-level `.figure-inline` with the image and the caption
  **directly below** at full content width.

### Theming tokens

CSS custom properties; light defaults, `@media (prefers-color-scheme: dark)` overrides.
Indigo accent, highlight = wash + inset underline (visible in both themes):

```
light: --bg#fcfcfd --fg#23232b --muted#6b6b78 --rule#e3e3ec
       --accent#5b4bd6 --accent-ink#4636c0 --hl-bg rgba(91,75,214,.14) --hl-ink#3c2fae
dark:  --bg#15151b --fg#e8e8f0 --muted#a0a0b0 --rule#2c2c39
       --accent#a99bff --accent-ink#c3b8ff --hl-bg rgba(169,155,255,.22) --hl-ink#d8d0ff
```
`html { color-scheme: light dark }` so form controls/scrollbars follow the theme.

## Files changed

- `internal/web/templates/base.tmpl` — drop the `tufte.css` `<link>`; keep `app.css`;
  ensure `<meta name="color-scheme">`/`color-scheme` CSS. Port the small base styles the
  collection page needs (body font, links, `.paper-list`, `.status-badge`) into `app.css`.
- `internal/web/templates/paper.tmpl` — rewrite to the claim-grid structure above. The
  `sidenotes`/`figurenotes` sub-templates become `.note`/`.figure-note` blocks (the
  Tufte `margin-toggle` checkbox hack is removed). `inlinefigures` stays for the
  implementation architecture figure (already section-level + caption-below).
- `internal/web/static/app.css` — rewrite as the Design C stylesheet (tokens + layout).
- `internal/web/static/tufte.css` — **removed**.
- `internal/web/view.go` — expected unchanged (verify the claim keys the template looks
  up match what `BuildPaperView` produces). No new fields anticipated.
- Tests: `render_test.go` / `handlers_test.go` — update class/markup expectations
  (e.g. assert `.claim`, `.claim-notes`, `.note`, `.hl`, outline numbers/levels) and drop
  assertions tied to removed Tufte markup.

## Optional / deferred

- Pure-CSS click-to-zoom lightbox for the bounded thumbnails (the existing JS-free
  `:checked` lightbox could be ported). Deferred unless wanted — thumbnails + caption are
  enough for v1.

## Testing

- Keep/extend `view.go` unit tests (outline number/level, figure-label matching, caption
  dedupe — all already passing).
- Update render tests to assert the new structure renders for each card section and that
  a claim with evidence emits a `.claim-notes` with `.note` + markers.
- Manual/visual verification via a local web binary + browser screenshot at desktop and
  narrow widths, in light and dark, confirming: no image overflow, aligned columns,
  legible dark-mode highlights. (Mockup "Design C" is the visual reference.)

## Reference

Approved visual mockup: "Design C · Editorial" (built during brainstorming;
serif headings + sans body, indigo accent, divided two-column, OS-driven theme).
