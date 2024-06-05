package github

import (
	"strings"

	"github.com/google/go-github/v61/github"
	"github.com/jamestelfer/chinmina-bridge/internal/config"
)

type Permissions struct {
	Permissions []string
}

func (p Permissions) ToGithubPermissions() *github.InstallationPermissions {
	permissions := &github.InstallationPermissions{}
	permMap := map[string]**string{
		"actions:read":              &permissions.Actions,
		"actions:write":             &permissions.Actions,
		"checks:read":               &permissions.Checks,
		"checks:write":              &permissions.Checks,
		"contents:read":             &permissions.Contents,
		"contents:write":            &permissions.Contents,
		"deployments:read":          &permissions.Deployments,
		"deployments:write":         &permissions.Deployments,
		"issues:read":               &permissions.Issues,
		"issues:write":              &permissions.Issues,
		"metadata:read":             &permissions.Metadata,
		"metadata:write":            &permissions.Metadata,
		"packages:read":             &permissions.Packages,
		"packages:write":            &permissions.Packages,
		"pages:read":                &permissions.Pages,
		"pages:write":               &permissions.Pages,
		"pull_requests:read":        &permissions.PullRequests,
		"pull_requests:write":       &permissions.PullRequests,
		"repository_projects:read":  &permissions.RepositoryProjects,
		"repository_projects:write": &permissions.RepositoryProjects,
		"security_events:read":      &permissions.SecurityEvents,
		"security_events:write":     &permissions.SecurityEvents,
		"statuses:read":             &permissions.Statuses,
		"statuses:write":            &permissions.Statuses,
	}

	for _, perm := range p.Permissions {
		if field, ok := permMap[perm]; ok {
			setPermission(field, perm)
		}
	}

	return permissions
}

func GetPermissionsFromConfig(config *config.Configuration, repoName string) (Permissions, error) {
	configPerms, err := config.GetPermissionsFromConfig(config, repoName)
	if err != nil {
		return Permissions{}, err
	}

	return Permissions{
		Permissions: configPerms,
	}, nil
}

func setPermission(field **string, value string) {
	parts := strings.Split(value, ":")
	if len(parts) == 2 {
		*field = github.String(parts[1])
	}
}
