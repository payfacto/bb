package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func DeploymentListString(deployments []bitbucket.Deployment) string {
	if len(deployments) == 0 {
		return "No deployments found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-38s", "ENV UUID")),
		LabelStyle.Render(fmt.Sprintf("%-30s", "STATE")),
		LabelStyle.Render(fmt.Sprintf("%-8s", "COMMIT")),
		LabelStyle.Render("DATE"))
	divider := fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 38)),
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 8)),
		SepStyle.Render(strings.Repeat("─", 10)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, d := range deployments {
		status := d.State.Name
		if d.State.Status != nil {
			status += "/" + d.State.Status.Name
		}
		commit := ""
		if d.Deployable.Commit != nil && len(d.Deployable.Commit.Hash) >= shortHashLen {
			commit = d.Deployable.Commit.Hash[:shortHashLen]
		}
		date := ""
		if len(d.LastUpdateTime) >= datePrefixLen {
			date = d.LastUpdateTime[:datePrefixLen]
		}
		sb.WriteString(fmt.Sprintf("  %-38s  %-30s  %s  %s\n",
			truncate(d.Environment.UUID, 38), StateBadge(status), IDStyle.Render(commit), DimStyle.Render(date)))
	}
	return sb.String()
}

func DeploymentList(deployments []bitbucket.Deployment) { fmt.Print(DeploymentListString(deployments)) }

func EnvListString(envs []bitbucket.Environment) string {
	if len(envs) == 0 {
		return "No environments found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-38s", "UUID")),
		LabelStyle.Render(fmt.Sprintf("%-12s", "TYPE")),
		LabelStyle.Render(fmt.Sprintf("%-20s", "NAME")),
		LabelStyle.Render("LOCK"))
	divider := fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 38)),
		SepStyle.Render(strings.Repeat("─", 12)),
		SepStyle.Render(strings.Repeat("─", 20)),
		SepStyle.Render(strings.Repeat("─", 8)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, e := range envs {
		lock := ""
		if e.Lock.Name == "LOCKED" {
			lock = StateBadge("LOCKED")
		}
		sb.WriteString(fmt.Sprintf("  %-38s  %-12s  %-20s  %s\n",
			e.UUID, e.EnvironmentType.Name, e.Name, lock))
	}
	return sb.String()
}

func EnvList(envs []bitbucket.Environment) { fmt.Print(EnvListString(envs)) }

func WebhookListString(hooks []bitbucket.Webhook) string {
	if len(hooks) == 0 {
		return "No webhooks found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-38s", "UUID")),
		LabelStyle.Render(fmt.Sprintf("%-8s", "ACTIVE")),
		LabelStyle.Render("URL"))
	divider := fmt.Sprintf("  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 38)),
		SepStyle.Render(strings.Repeat("─", 8)),
		SepStyle.Render(strings.Repeat("─", 40)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, h := range hooks {
		active := DimStyle.Render("false")
		if h.Active {
			active = StateBadge("true")
		}
		sb.WriteString(fmt.Sprintf("  %-38s  %-8s  %s\n", h.UUID, active, truncate(h.URL, 60)))
	}
	return sb.String()
}

func WebhookList(hooks []bitbucket.Webhook) { fmt.Print(WebhookListString(hooks)) }

func DeployKeyListString(keys []bitbucket.DeployKey) string {
	if len(keys) == 0 {
		return "No deploy keys found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-6s", "ID")),
		LabelStyle.Render(fmt.Sprintf("%-30s", "LABEL")),
		LabelStyle.Render("KEY"))
	divider := fmt.Sprintf("  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 6)),
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 40)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("  %s  %-30s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-5d", k.ID)), truncate(k.Label, 30), truncate(k.Key, 50)))
	}
	return sb.String()
}

func DeployKeyList(keys []bitbucket.DeployKey) { fmt.Print(DeployKeyListString(keys)) }

func DownloadListString(downloads []bitbucket.Download) string {
	if len(downloads) == 0 {
		return "No downloads found.\n"
	}
	header := fmt.Sprintf("  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-40s", "NAME")),
		LabelStyle.Render("SIZE"))
	divider := fmt.Sprintf("  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 40)),
		SepStyle.Render(strings.Repeat("─", 12)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, d := range downloads {
		sb.WriteString(fmt.Sprintf("  %-40s  %s\n",
			truncate(d.Name, 40), DimStyle.Render(fmt.Sprintf("%d bytes", d.Size))))
	}
	return sb.String()
}

func DownloadList(downloads []bitbucket.Download) { fmt.Print(DownloadListString(downloads)) }

func RestrictionListString(restrictions []bitbucket.BranchRestriction) string {
	if len(restrictions) == 0 {
		return "No branch restrictions found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-6s", "ID")),
		LabelStyle.Render(fmt.Sprintf("%-40s", "KIND")),
		LabelStyle.Render(fmt.Sprintf("%-6s", "VALUE")),
		LabelStyle.Render("PATTERN"))
	divider := fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 6)),
		SepStyle.Render(strings.Repeat("─", 40)),
		SepStyle.Render(strings.Repeat("─", 6)),
		SepStyle.Render(strings.Repeat("─", 20)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, r := range restrictions {
		valueStr := ""
		if r.Value != nil {
			valueStr = fmt.Sprintf("%d", *r.Value)
		}
		sb.WriteString(fmt.Sprintf("  %s  %-40s  %-6s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-5d", r.ID)), truncate(r.Kind, 40), valueStr, r.Pattern))
	}
	return sb.String()
}

func RestrictionList(restrictions []bitbucket.BranchRestriction) {
	fmt.Print(RestrictionListString(restrictions))
}

func MemberListString(members []bitbucket.WorkspaceMember) string {
	if len(members) == 0 {
		return "No members found.\n"
	}
	header := fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-30s", "NAME")),
		LabelStyle.Render(fmt.Sprintf("%-20s", "NICKNAME")),
		LabelStyle.Render("ACCOUNT ID"))
	divider := fmt.Sprintf("  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 20)),
		SepStyle.Render(strings.Repeat("─", 20)))
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)
	for _, m := range members {
		sb.WriteString(fmt.Sprintf("  %-30s  %-20s  %s\n",
			truncate(m.User.DisplayName, 30), m.User.Nickname, DimStyle.Render(m.User.AccountID)))
	}
	return sb.String()
}

func MemberList(members []bitbucket.WorkspaceMember) { fmt.Print(MemberListString(members)) }

func UserMeString(u bitbucket.User) string {
	const labelW = 10
	label := func(s string) string {
		return LabelStyle.Render(fmt.Sprintf("  %-*s", labelW, s))
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Name"), u.DisplayName))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Nickname"), "@"+u.Nickname))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Account"), DimStyle.Render(u.AccountID)))
	if u.Links.HTML.Href != "" {
		sb.WriteString(fmt.Sprintf("%s  %s\n", label("Profile"), u.Links.HTML.Href))
	}
	return sb.String()
}

func UserMe(u bitbucket.User) { fmt.Print(UserMeString(u)) }
