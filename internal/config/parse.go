package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseConfiguration(data []byte) (*Configuration, error) {
	config := NewDefaultConfig()
	err := yaml.Unmarshal([]byte(data), config)
	if err != nil {
		return nil, err
	}
	sanitizeConfig(config)
	return config, nil
}

func sanitizeConfig(config *Configuration) {
	for i := range config.Default.Permissions {
		config.Default.Permissions[i] = strings.ToLower(config.Default.Permissions[i])
	}

	for i := range config.Default.Repositories {
		for j := range config.Default.Repositories[i].Permissions {
			config.Default.Repositories[i].Permissions[j] = strings.ToLower(
				config.Default.Repositories[i].Permissions[j],
			)
		}
	}

	for i := range config.Pipelines {
		for j := range config.Pipelines[i].Repositories {
			for k := range config.Pipelines[i].Repositories[j].Permissions {
				config.Pipelines[i].Repositories[j].Permissions[k] = strings.ToLower(
					config.Pipelines[i].Repositories[j].Permissions[k],
				)
			}
		}
	}
}

func GetConfigValue(config *Configuration, repoName string, defaultValue []string,
	getValue func(repo Repository) []string) ([]string, error) {
	for _, repo := range config.Default.Repositories {
		if repo.Name == repoName {
			return mergeValues(defaultValue, getValue(repo)), nil
		}
	}

	for _, pipeline := range config.Pipelines {
		for _, repo := range pipeline.Repositories {
			if repo.Name == repoName {
				return mergeValues(defaultValue, getValue(repo)), nil
			}
		}
	}

	return nil, fmt.Errorf("repository %s not found in configuration", repoName)
}

func mergeValues(defaultValues, repoValues []string) []string {
	valueSet := make(map[string]struct{})

	for _, value := range defaultValues {
		valueSet[value] = struct{}{}
	}

	for _, value := range repoValues {
		valueSet[value] = struct{}{}
	}

	merged := make([]string, 0, len(valueSet))
	for value := range valueSet {
		merged = append(merged, value)
	}

	return merged
}
