package github

import (
	"fmt"
	"os"

	"github.com/google/go-github/v61/github"
)

type Permissions struct {
	Actions            string
	Checks             string
	Contents           string
	Deployments        string
	Issues             string
	Metadata           string
	Packages           string
	Pages              string
	PullRequests       string
	RepositoryProjects string
	SecurityEvents     string
	Statuses           string
}

func (p Permissions) ToGithubPermissions() *github.InstallationPermissions {
	permissions := &github.InstallationPermissions{}

	setPermission(&permissions.Actions, p.Actions)
	setPermission(&permissions.Checks, p.Checks)
	setPermission(&permissions.Contents, p.Contents)
	setPermission(&permissions.Deployments, p.Deployments)
	setPermission(&permissions.Issues, p.Issues)
	setPermission(&permissions.Metadata, p.Metadata)
	setPermission(&permissions.Packages, p.Packages)
	setPermission(&permissions.Pages, p.Pages)
	setPermission(&permissions.PullRequests, p.PullRequests)
	setPermission(&permissions.RepositoryProjects, p.RepositoryProjects)
	setPermission(&permissions.SecurityEvents, p.SecurityEvents)
	setPermission(&permissions.Statuses, p.Statuses)

	return permissions
}

// GetPermissionsFromEnv retrieves GitHub token permissions from Buildkite
// environment variables. It reads the permissions for various GitHub features
// from environment variables that should be set to either "read" or "write". If
// an environment variable is not set, the permission is omitted.
func GetPermissionsFromEnv() (Permissions, error) {
	permissions := Permissions{}

	envVars := map[string]*string{
		"GITHUB_PERMISSIONS_ACTIONS":             &permissions.Actions,
		"GITHUB_PERMISSIONS_CHECKS":              &permissions.Checks,
		"GITHUB_PERMISSIONS_CONTENTS":            &permissions.Contents,
		"GITHUB_PERMISSIONS_DEPLOYMENTS":         &permissions.Deployments,
		"GITHUB_PERMISSIONS_ISSUES":              &permissions.Issues,
		"GITHUB_PERMISSIONS_METADATA":            &permissions.Metadata,
		"GITHUB_PERMISSIONS_PACKAGES":            &permissions.Packages,
		"GITHUB_PERMISSIONS_PAGES":               &permissions.Pages,
		"GITHUB_PERMISSIONS_PULL_REQUESTS":       &permissions.PullRequests,
		"GITHUB_PERMISSIONS_REPOSITORY_PROJECTS": &permissions.RepositoryProjects,
		"GITHUB_PERMISSIONS_SECURITY_EVENTS":     &permissions.SecurityEvents,
		"GITHUB_PERMISSIONS_STATUSES":            &permissions.Statuses,
	}

	defaultValues := map[string]string{
		"GITHUB_PERMISSIONS_CONTENTS": "read",
	}

	for envVar, field := range envVars {
		defaultValue := defaultValues[envVar]
		value := getEnv(envVar, defaultValue)
		err := validatePermission(value, envVar)
		if err != nil {
			return permissions, err
		}
		*field = value
	}

	return permissions, nil
}

func setPermission(field **string, value string) {
	if value != "" {
		*field = github.String(value)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

var validPermissions = map[string]struct{}{
	"read":  {},
	"write": {},
}

func validatePermission(permission, envVar string) error {
	if permission == "" {
		return nil
	}
	if _, valid := validPermissions[permission]; !valid {
		return fmt.Errorf("invalid value for %s: %s", envVar, permission)
	}
	return nil
}
