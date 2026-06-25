package web

import (
	"fmt"
	"strings"

	"scholarflow_web/internal/apiclient"
)

// OutlineEntry is one parsed-section heading for the thesis outline.
type OutlineEntry struct {
	Heading string
	Page    *int
}

// PaperView is the reading-page model. Evidence and figure callouts are grouped
// by a composite claim key so each claim — or each individual bullet of a list
// field — renders its own sidenotes and inline figure callouts.
type PaperView struct {
	Detail          apiclient.PaperDetail
	Outline         []OutlineEntry
	EvidenceByClaim map[string][]EvidenceNote
	FiguresByClaim  map[string][]FigureNote
}

// EvidenceNote is one rendered sidenote. DOMID is unique across the page so the
// Tufte checkbox toggles don't collide between bullets that share a claim key.
type EvidenceNote struct {
	DOMID     string
	Page      *int
	SectionID string
	Snippet   string
}

// FigureNote is one inline figure callout placed at a claim anchor.
type FigureNote struct {
	DOMID   string
	Label   string
	Page    *int
	Caption string
}

// claimKey is the grouping/lookup key: the field name for scalar fields, or
// "field#N" for the 0-based bullet N of a list field. Mirrors the template's
// printf "field#%d" lookups.
func claimKey(field string, index *int) string {
	if index == nil {
		return field
	}
	return fmt.Sprintf("%s#%d", field, *index)
}

// BuildPaperView groups a paper's card evidence and figure placements for
// rendering. Figure captions are resolved from PaperDetail.Figures by label.
func BuildPaperView(detail apiclient.PaperDetail) PaperView {
	view := PaperView{Detail: detail}
	for _, s := range detail.Sections {
		if s.Heading == nil || strings.TrimSpace(*s.Heading) == "" {
			continue
		}
		view.Outline = append(view.Outline, OutlineEntry{
			Heading: *s.Heading,
			Page:    intFromInt32(s.PageStart),
		})
	}
	if detail.Card == nil {
		return view
	}

	id := 0
	view.EvidenceByClaim = make(map[string][]EvidenceNote)
	for _, e := range detail.Card.Evidence {
		id++
		key := claimKey(e.ClaimKey, e.ClaimIndex)
		view.EvidenceByClaim[key] = append(view.EvidenceByClaim[key], EvidenceNote{
			DOMID:     fmt.Sprintf("sn-%d", id),
			Page:      e.Page,
			SectionID: e.SectionID,
			Snippet:   e.Snippet,
		})
	}

	captionByLabel := make(map[string]string, len(detail.Figures))
	for _, f := range detail.Figures {
		captionByLabel[normalizeLabel(f.Label)] = f.Caption
	}
	view.FiguresByClaim = make(map[string][]FigureNote)
	for _, f := range detail.Card.Figures {
		id++
		key := claimKey(f.ClaimKey, f.ClaimIndex)
		view.FiguresByClaim[key] = append(view.FiguresByClaim[key], FigureNote{
			DOMID:   fmt.Sprintf("fn-%d", id),
			Label:   f.Label,
			Page:    f.Page,
			Caption: captionByLabel[normalizeLabel(f.Label)],
		})
	}
	return view
}

func normalizeLabel(label string) string {
	return strings.Join(strings.Fields(strings.ToLower(label)), " ")
}

func intFromInt32(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
