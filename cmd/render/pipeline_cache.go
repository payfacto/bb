package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	cacheNameWidth = 24
	cacheUUIDWidth = 38
	cacheSizeWidth = 10
	bytesPerUnit   = 1024
)

// HumanBytes formats a byte count with binary (1024-based) units, one decimal
// place above bytes: 1536 -> "1.5 KiB".
func HumanBytes(n int64) string {
	if n < bytesPerUnit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(n) / bytesPerUnit
	unit := 0
	for value >= bytesPerUnit && unit < len(units)-1 {
		value /= bytesPerUnit
		unit++
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

// PipelineCacheListString returns formatted text for a list of pipeline caches.
func PipelineCacheListString(caches []bitbucket.PipelineCache) string {
	if len(caches) == 0 {
		return "No pipeline caches found.\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "  %s  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-*s", cacheNameWidth, "NAME")),
		LabelStyle.Render(fmt.Sprintf("%-*s", cacheUUIDWidth, "UUID")),
		LabelStyle.Render(fmt.Sprintf("%-*s", cacheSizeWidth, "SIZE")),
		LabelStyle.Render(fmt.Sprintf("%-*s", datePrefixLen, "CREATED")),
		LabelStyle.Render("PATH"))
	fmt.Fprintf(&sb, "  %s  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", cacheNameWidth)),
		SepStyle.Render(strings.Repeat("─", cacheUUIDWidth)),
		SepStyle.Render(strings.Repeat("─", cacheSizeWidth)),
		SepStyle.Render(strings.Repeat("─", datePrefixLen)),
		SepStyle.Render(strings.Repeat("─", 4)))
	for _, c := range caches {
		created := c.CreatedOn
		if len(created) >= datePrefixLen {
			created = created[:datePrefixLen]
		}
		fmt.Fprintf(&sb, "  %s  %s  %-*s  %s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-*s", cacheNameWidth, truncate(c.Name, cacheNameWidth))),
			DimStyle.Render(fmt.Sprintf("%-*s", cacheUUIDWidth, c.UUID)),
			cacheSizeWidth, HumanBytes(c.FileSizeBytes),
			DimStyle.Render(fmt.Sprintf("%-*s", datePrefixLen, created)),
			c.Path)
	}
	return sb.String()
}

// PipelineCacheList prints the formatted cache list to stdout.
func PipelineCacheList(caches []bitbucket.PipelineCache) { fmt.Print(PipelineCacheListString(caches)) }
