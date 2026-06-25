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
