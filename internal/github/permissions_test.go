package github_test

import (
	"os"
	"testing"

	gh "github.com/google/go-github/v61/github"
	"github.com/jamestelfer/chinmina-bridge/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPermissionsFromEnv(t *testing.T) {
	// set environment variables for testing
	os.Setenv("GITHUB_PERMISSIONS_ACTIONS", "read")
	os.Setenv("GITHUB_PERMISSIONS_CHECKS", "read")
	os.Setenv("GITHUB_PERMISSIONS_CONTENTS", "write") // overriding default
	os.Setenv("GITHUB_PERMISSIONS_DEPLOYMENTS", "read")
	os.Setenv("GITHUB_PERMISSIONS_ISSUES", "read")
	os.Setenv("GITHUB_PERMISSIONS_METADATA", "read")
	os.Setenv("GITHUB_PERMISSIONS_PACKAGES", "read")
	os.Setenv("GITHUB_PERMISSIONS_PAGES", "read")
	os.Setenv("GITHUB_PERMISSIONS_PULL_REQUESTS", "read")
	os.Setenv("GITHUB_PERMISSIONS_REPOSITORY_PROJECTS", "read")
	os.Setenv("GITHUB_PERMISSIONS_SECURITY_EVENTS", "read")
	os.Setenv("GITHUB_PERMISSIONS_STATUSES", "read")

	permissions, err := github.GetPermissionsFromEnv()
	require.NoError(t, err)

	assert.Equal(t, "read", permissions.Actions)
	assert.Equal(t, "read", permissions.Checks)
	assert.Equal(t, "write", permissions.Contents)
	assert.Equal(t, "read", permissions.Deployments)
	assert.Equal(t, "read", permissions.Issues)
	assert.Equal(t, "read", permissions.Metadata)
	assert.Equal(t, "read", permissions.Packages)
	assert.Equal(t, "read", permissions.Pages)
	assert.Equal(t, "read", permissions.PullRequests)
	assert.Equal(t, "read", permissions.RepositoryProjects)
	assert.Equal(t, "read", permissions.SecurityEvents)
	assert.Equal(t, "read", permissions.Statuses)

	os.Clearenv()
}

func TestToGithubPermissions(t *testing.T) {
	permissions := github.Permissions{
		Actions:            "read",
		Checks:             "read",
		Contents:           "write",
		Deployments:        "read",
		Issues:             "read",
		Metadata:           "read",
		Packages:           "read",
		Pages:              "read",
		PullRequests:       "read",
		RepositoryProjects: "read",
		SecurityEvents:     "read",
		Statuses:           "read",
	}

	githubPermissions := permissions.ToGithubPermissions()

	assert.Equal(t, gh.String("read"), githubPermissions.Actions)
	assert.Equal(t, gh.String("read"), githubPermissions.Checks)
	assert.Equal(t, gh.String("write"), githubPermissions.Contents)
	assert.Equal(t, gh.String("read"), githubPermissions.Deployments)
	assert.Equal(t, gh.String("read"), githubPermissions.Issues)
	assert.Equal(t, gh.String("read"), githubPermissions.Metadata)
	assert.Equal(t, gh.String("read"), githubPermissions.Packages)
	assert.Equal(t, gh.String("read"), githubPermissions.Pages)
	assert.Equal(t, gh.String("read"), githubPermissions.PullRequests)
	assert.Equal(t, gh.String("read"), githubPermissions.RepositoryProjects)
	assert.Equal(t, gh.String("read"), githubPermissions.SecurityEvents)
	assert.Equal(t, gh.String("read"), githubPermissions.Statuses)
}
