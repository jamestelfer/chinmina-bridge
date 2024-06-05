package config

func NewDefaultConfig() *Configuration {
	return &Configuration{
		Default: DefaultConfig{
			Permissions:  []string{},
			Repositories: []Repository{},
		},
		Pipelines: []Pipeline{},
	}
}
