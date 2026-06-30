package web

import (
	"testing"

	"scholarflow_web/internal/apiclient"
)

func i32(v int32) *int32  { return &v }
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
