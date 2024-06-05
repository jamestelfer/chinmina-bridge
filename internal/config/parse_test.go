package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const configStr string = `
default:
  permissions:
    - "content:read"
  repositories:
    - name: "org/privaterepo"
      permissions: ["content:read"]
    - name: "org/otherrepo"

pipelines:
  - name: "pipeline name"
    repositories:
      - name: "org/repo1"
        permissions: ["content:read", "pull_request:write"]
      - name: "org/repo2"
        permissions: ["content:read", "pull_request:write"]
`

func TestParseConfiguration(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		data     string
		expected *Configuration
	}{
		{
			name: "default values",
			data: "",
			expected: &Configuration{
				Default: DefaultConfig{
					Permissions:  nil,
					Repositories: nil,
				},
				Pipelines: nil,
			},
		},
		{
			name: "user provided values",
			data: configStr,
			expected: &Configuration{
				Default: DefaultConfig{
					Permissions: []string{"content:read"},
					Repositories: []Repository{
						{Name: "org/privaterepo", Permissions: []string{"content:read"}},
						{Name: "org/otherrepo"},
					},
				},
				Pipelines: []Pipeline{
					{
						Name: "pipeline name",
						Repositories: []Repository{
							{Name: "org/repo1", Permissions: []string{"content:read", "pull_request:write"}},
							{Name: "org/repo2", Permissions: []string{"content:read", "pull_request:write"}},
						},
					},
				},
			},
		},
		{
			name: "user provided empty default permissions",
			data: `default:
  permissions: []
pipelines: []`,
			expected: &Configuration{
				Default: DefaultConfig{
					Permissions:  []string{},
					Repositories: nil,
				},
				Pipelines: []Pipeline{},
			},
		},
		{
			name: "user provided empty repositories",
			data: `default:
  permissions: []
  repositories: []
pipelines: []`,
			expected: &Configuration{
				Default: DefaultConfig{
					Permissions:  []string{},
					Repositories: []Repository{},
				},
				Pipelines: []Pipeline{},
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			config, err := ParseConfiguration([]byte(scenario.data))
			require.NoError(t, err)
			assert.Equal(t, scenario.expected.Default, config.Default)
			assert.Equal(t, scenario.expected.Pipelines, config.Pipelines)
		})
	}
}
