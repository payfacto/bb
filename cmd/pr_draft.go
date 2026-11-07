package cmd

import (
	"fmt"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// draftFilter selects PRs by draft state in `bb pr list`.
type draftFilter int

const (
	draftAny     draftFilter = iota // no filter
	draftOnly                       // --draft
	draftExclude                    // --no-draft
)

// filterPRsByDraft keeps the PRs matching filter. Bitbucket's list endpoint
// has no documented draft query, so the filter runs client-side after the
// fetch.
func filterPRsByDraft(prs []bitbucket.PR, filter draftFilter) []bitbucket.PR {
	if filter == draftAny {
		return prs
	}
	wantDraft := filter == draftOnly
	kept := make([]bitbucket.PR, 0, len(prs))
	for _, pr := range prs {
		if pr.Draft == wantDraft {
			kept = append(kept, pr)
		}
	}
	return kept
}

// ensureMergeable rejects a draft PR with a clear, actionable error instead of
// passing Bitbucket's merge failure through.
func ensureMergeable(pr bitbucket.PR) error {
	if !pr.Draft {
		return nil
	}
	return newCLIError(ErrCodeValidationFailed,
		fmt.Sprintf("PR #%d is a draft; mark it ready first (bb pr update -p %d --ready)", pr.ID, pr.ID), nil)
}
