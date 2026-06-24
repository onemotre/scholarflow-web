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
				{ClaimKey: "method", SectionID: "3", Snippet: "a"},
				{ClaimKey: "results", ClaimIndex: intPtr(0), SectionID: "4", Snippet: "b"},
				{ClaimKey: "results", ClaimIndex: intPtr(0), SectionID: "5", Snippet: "c"},
			},
			Figures: []apiclient.CardFigure{
				{Label: "figure 2", ClaimKey: "results", ClaimIndex: intPtr(0), Page: intPtr(3)},
			},
		},
	})
	if len(view.EvidenceByClaim["method"]) != 1 {
		t.Fatalf("method evidence = %#v", view.EvidenceByClaim["method"])
	}
	if len(view.EvidenceByClaim["results#0"]) != 2 {
		t.Fatalf("results#0 evidence = %#v", view.EvidenceByClaim["results#0"])
	}
	figs := view.FiguresByClaim["results#0"]
	if len(figs) != 1 || figs[0].Caption != "结构图" || figs[0].Page == nil || *figs[0].Page != 3 {
		t.Fatalf("results#0 figures = %#v", figs)
	}
	// DOM ids must be unique across all notes.
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

func TestRenderPaperWithSidenotes(t *testing.T) {
	method := "我们提出 TD-MPC 方法"
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Card: &apiclient.Card{
			Method:   method,
			Evidence: []apiclient.Evidence{{ClaimKey: "method", SectionID: "3", Snippet: "证据片段"}},
		},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{method, "sidenote", "[§3]", "证据片段"} {
		if !strings.Contains(out, want) {
			t.Fatalf("paper missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderPaperResultsPerBulletWithPagesAndCommas(t *testing.T) {
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Figures: []apiclient.Figure{{Label: "Figure 2", Caption: "成功率曲线"}},
		Card: &apiclient.Card{
			Results: []string{"结果一", "结果二"},
			Method:  "方法说明",
			Evidence: []apiclient.Evidence{
				{ClaimKey: "results", ClaimIndex: intPtr(0), Page: intPtr(7), Snippet: "证据A"},
				{ClaimKey: "results", ClaimIndex: intPtr(0), Page: intPtr(8), Snippet: "证据B"},
			},
			Figures: []apiclient.CardFigure{
				{Label: "Figure 2", ClaimKey: "method", Page: intPtr(5)},
			},
		},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	// Per-bullet results evidence with page labels.
	for _, want := range []string{"结果一", "结果二", "[p.7]", "[p.8]", "证据A", "证据B"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// Adjacent markers comma-separated.
	if !strings.Contains(out, "sidenote-comma") {
		t.Fatalf("expected comma separator between adjacent sidenotes:\n%s", out)
	}
	// Figure callout placed inline at the method claim, with caption + page.
	for _, want := range []string{"figure-callout", "成功率曲线", "第 5 页"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing figure callout %q in:\n%s", want, out)
		}
	}
	// No standalone figure dump sections.
	for _, unwant := range []string{"关键图表", "图表标题"} {
		if strings.Contains(out, unwant) {
			t.Fatalf("did not expect dump section %q in:\n%s", unwant, out)
		}
	}
}

func TestRenderPaperBackwardCompatibleWithV1Card(t *testing.T) {
	// A 1.0 card has no claim_index / figures / page fields.
	var b strings.Builder
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
		Card: &apiclient.Card{
			Method:   "旧版方法",
			Results:  []string{"旧结果"},
			Evidence: []apiclient.Evidence{{ClaimKey: "method", SectionID: "2", Snippet: "旧证据"}},
		},
	})
	if err := Render(&b, "paper.tmpl", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{"旧版方法", "旧结果", "[§2]", "旧证据"} {
		if !strings.Contains(out, want) {
			t.Fatalf("v1 card missing %q in:\n%s", want, out)
		}
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
