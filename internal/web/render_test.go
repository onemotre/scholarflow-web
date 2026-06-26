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
}

func TestBaseUsesEditorialStylesheetAndKatex(t *testing.T) {
	var b strings.Builder
	if err := Render(&b, "collection.tmpl", []apiclient.PaperSummary(nil)); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if strings.Contains(out, "tufte.css") {
		t.Fatalf("base still links tufte.css:\n%s", out)
	}
	for _, want := range []string{"/static/app.css", `class="paper"`, "katex.min.css", "katex-init.js"} {
		if !strings.Contains(out, want) {
			t.Fatalf("base missing %q:\n%s", want, out)
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
		// claim grid with aligned evidence column:
		`class="claim"`, `class="claim-body"`, `class="claim-notes"`,
		`class="note"`, `class="marker"`, "§3", "p.7", "证据A",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("paper missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "margin-toggle") || strings.Contains(out, "sidenote") {
		t.Fatalf("old Tufte markup still present:\n%s", out)
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
			{Order: 1, Number: strPtr("1."), Heading: strPtr("Introduction"), PageStart: int32Ptr(1)},
			{Order: 2, Heading: nil},        // skipped (no heading)
			{Order: 3, Heading: strPtr("")}, // skipped (empty heading)
			{Order: 4, Number: strPtr("2."), Heading: strPtr("Results"), PageStart: int32Ptr(6)},
			{Order: 5, Number: strPtr("2.1."), Heading: strPtr("Motion Tracking"), PageStart: int32Ptr(6)},
			{Order: 6, Heading: strPtr("Run-in heading")}, // unnumbered -> nests under 2.1
		},
	})
	if len(view.Outline) != 4 {
		t.Fatalf("outline = %#v, want 4 entries", view.Outline)
	}
	if e := view.Outline[0]; e.Heading != "Introduction" || e.Number != "1" || e.Level != 0 || e.Page == nil || *e.Page != 1 {
		t.Fatalf("outline[0] = %#v", e)
	}
	if e := view.Outline[1]; e.Heading != "Results" || e.Number != "2" || e.Level != 0 {
		t.Fatalf("outline[1] = %#v", e)
	}
	if e := view.Outline[2]; e.Heading != "Motion Tracking" || e.Number != "2.1" || e.Level != 1 {
		t.Fatalf("outline[2] = %#v", e)
	}
	if e := view.Outline[3]; e.Number != "" || e.Level != 2 {
		t.Fatalf("outline[3] (unnumbered run-in) = %#v, want level 2", e)
	}
}

func TestBuildPaperViewMatchesNoisyFigureLabels(t *testing.T) {
	// Card labels are clean ("Figure 2"); GROBID detail labels carry a trailing
	// " :" ("Figure 2 :"). They must still resolve to the same figure.
	view := BuildPaperView(apiclient.PaperDetail{
		PaperID: "p1",
		Figures: []apiclient.Figure{{Label: "Figure 2 :", Caption: "曲线图", ID: "f2", HasImage: true}},
		Card: &apiclient.Card{
			Results: []apiclient.CardResult{{Metric: "acc"}},
			Figures: []apiclient.CardFigure{{Label: "Figure 2", ClaimKey: "results", ClaimIndex: intPtr(0)}},
		},
	})
	notes := view.FiguresByClaim["results#0"]
	if len(notes) != 1 {
		t.Fatalf("FiguresByClaim[results#0] = %#v, want 1 note", notes)
	}
	if n := notes[0]; !n.HasImage || n.Caption != "曲线图" || n.ImageURL != "/papers/p1/figures/f2/image" {
		t.Fatalf("note did not resolve noisy label: %#v", n)
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
	// Architecture figure renders inline in the body; chart figure renders in the
	// claim's notes column.
	if !strings.Contains(out, `class="figure-inline"`) || !strings.Contains(out, "/papers/p1/figures/f1/image") {
		t.Fatalf("architecture inline figure missing:\n%s", out)
	}
	if !strings.Contains(out, `class="figure-note"`) || !strings.Contains(out, "/papers/p1/figures/f3/image") {
		t.Fatalf("chart figure note missing:\n%s", out)
	}
}
