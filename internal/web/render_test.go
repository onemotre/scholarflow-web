package web

import (
	"strings"
	"testing"

	"scholarflow_web/internal/apiclient"
)

func TestGroupEvidenceByClaim(t *testing.T) {
	g := GroupEvidenceByClaim([]apiclient.Evidence{
		{ClaimKey: "method", SectionID: "3", Snippet: "a"},
		{ClaimKey: "method", SectionID: "4", Snippet: "b"},
		{ClaimKey: "results", SectionID: "5", Snippet: "c"},
	})
	if len(g["method"]) != 2 || len(g["results"]) != 1 {
		t.Fatalf("grouping = %#v", g)
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
	view := PaperView{
		Detail: apiclient.PaperDetail{
			PaperID: "p1", Status: "completed", UploadedFilename: "a.pdf",
			Card: &apiclient.Card{Method: method},
		},
		EvidenceByClaim: GroupEvidenceByClaim([]apiclient.Evidence{
			{ClaimKey: "method", SectionID: "3", Snippet: "证据片段"},
		}),
	}
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
