package web

import "scholarflow_web/internal/apiclient"

// PaperView is the reading-page model: the detail plus evidence grouped by the
// card field (claim_key) it supports, so each claim renders its own sidenotes.
type PaperView struct {
	Detail          apiclient.PaperDetail
	EvidenceByClaim map[string][]apiclient.Evidence
}

// GroupEvidenceByClaim buckets evidence entries by their ClaimKey.
func GroupEvidenceByClaim(evidence []apiclient.Evidence) map[string][]apiclient.Evidence {
	grouped := make(map[string][]apiclient.Evidence)
	for _, e := range evidence {
		grouped[e.ClaimKey] = append(grouped[e.ClaimKey], e)
	}
	return grouped
}
