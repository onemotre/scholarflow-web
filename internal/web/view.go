package web

import (
	"fmt"
	"strings"
	"unicode"

	"scholarflow_web/internal/apiclient"
)

// OutlineEntry is one parsed-section heading for the thesis outline. Number is
// the GROBID section number (e.g. "2.1") and Level is its 0-based indent depth.
type OutlineEntry struct {
	Number  string
	Heading string
	Page    *int
	Level   int
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

// EvidenceNote is one rendered evidence note in a claim's notes column.
type EvidenceNote struct {
	Page      *int
	SectionID string
	Snippet   string
}

// FigureNote is one figure placed at a claim anchor.
type FigureNote struct {
	Label    string
	Page     *int
	Caption  string
	HasImage bool
	ImageURL string
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
	lastLevel := 0
	for _, s := range detail.Sections {
		if s.Heading == nil || strings.TrimSpace(*s.Heading) == "" {
			continue
		}
		number := ""
		if s.Number != nil {
			number = strings.TrimRight(strings.TrimSpace(*s.Number), ".")
		}
		// Level comes from the section number's dot-depth (1 -> 0, 2.1 -> 1).
		// Unnumbered run-in subheadings nest one level under the last numbered one.
		level := lastLevel + 1
		if number != "" {
			level = strings.Count(number, ".")
			lastLevel = level
		}
		view.Outline = append(view.Outline, OutlineEntry{
			Number:  number,
			Heading: strings.TrimSpace(*s.Heading),
			Page:    intFromInt32(s.PageStart),
			Level:   level,
		})
	}
	if detail.Card == nil {
		return view
	}

	view.EvidenceByClaim = make(map[string][]EvidenceNote)
	for _, e := range detail.Card.Evidence {
		key := claimKey(e.ClaimKey, e.ClaimIndex)
		view.EvidenceByClaim[key] = append(view.EvidenceByClaim[key], EvidenceNote{
			Page:      e.Page,
			SectionID: e.SectionID,
			Snippet:   e.Snippet,
		})
	}

	figByLabel := make(map[string]apiclient.Figure, len(detail.Figures))
	for _, f := range detail.Figures {
		figByLabel[normalizeLabel(f.Label)] = f
	}
	view.FiguresByClaim = make(map[string][]FigureNote)
	for _, f := range detail.Card.Figures {
		key := claimKey(f.ClaimKey, f.ClaimIndex)
		src := figByLabel[normalizeLabel(f.Label)]
		note := FigureNote{
			Label:    f.Label,
			Page:     f.Page,
			Caption:  trimLabelPrefix(src.Caption, f.Label),
			HasImage: src.HasImage,
		}
		if src.HasImage {
			note.ImageURL = "/papers/" + detail.PaperID + "/figures/" + src.ID + "/image"
		}
		view.FiguresByClaim[key] = append(view.FiguresByClaim[key], note)
	}
	return view
}

// normalizeLabel reduces a figure label to its alphanumeric core so the card's
// clean label ("Figure 2") matches GROBID's noisier one ("Figure 2 :"). Both
// collapse to "figure2".
// trimLabelPrefix drops a leading repetition of the figure label from the
// caption (GROBID captions often start "Figure 7: ...") so the callout doesn't
// render "Figure 7：Figure 7: ...". Best-effort: only trims an exact prefix.
func trimLabelPrefix(caption, label string) string {
	c := strings.TrimSpace(caption)
	label = strings.TrimSpace(label)
	if label != "" && strings.HasPrefix(strings.ToLower(c), strings.ToLower(label)) {
		c = strings.TrimLeft(c[len(label):], " :：")
	}
	return c
}

func normalizeLabel(label string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(label) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func intFromInt32(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
