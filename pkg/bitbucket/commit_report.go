package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// reportsPath is the Code Insights reports collection for a commit.
func (r *CommitResource) reportsPath(hash string) string {
	return fmt.Sprintf("%s/commit/%s/reports", repoPath(r.workspace, r.repo), url.PathEscape(hash))
}

func (r *CommitResource) reportPath(hash, reportID string) string {
	return r.reportsPath(hash) + "/" + url.PathEscape(reportID)
}

// Reports returns the Code Insights reports (coverage, security scans, test
// results, ...) attached to a commit.
func (r *CommitResource) Reports(ctx context.Context, hash string) ([]Report, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[Report](ctx, r.client, r.reportsPath(hash), q)
}

// Report returns a single Code Insights report. reportID is either the
// report's UUID or the external_id its creator assigned.
func (r *CommitResource) Report(ctx context.Context, hash, reportID string) (Report, error) {
	data, err := r.client.do(ctx, "GET", r.reportPath(hash, reportID), nil, nil)
	if err != nil {
		return Report{}, err
	}
	return decode[Report](data)
}

// ReportAnnotations returns the individual findings recorded on a report.
func (r *CommitResource) ReportAnnotations(ctx context.Context, hash, reportID string) ([]ReportAnnotation, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[ReportAnnotation](ctx, r.client, r.reportPath(hash, reportID)+"/annotations", q)
}
