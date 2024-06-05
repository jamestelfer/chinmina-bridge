package config

import (
	"context"
	"fmt"
)

type PermissionsService struct {
	config *Configuration
}

func NewPermissionsService(config *Configuration) *PermissionsService {
	return &PermissionsService{config: config}
}

func (ps *PermissionsService) GetPermissions(ctx context.Context, pipelineName string) (Permissions, error) {
	permissions, err := GetPermissionsFromConfig(ps.config, pipelineName)
	if err != nil {
		return Permissions{}, fmt.Errorf("could not get permissions for pipeline %s: %w", pipelineName, err)
	}
	return Permissions{Permissions: permissions}, nil
}

func GetPermissionsFromConfig(config *Configuration, repoName string) ([]string, error) {
	return GetConfigValue(config, repoName, config.Default.Permissions, func(repo Repository) []string {
		return repo.Permissions
	})
}
