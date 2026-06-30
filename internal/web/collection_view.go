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
