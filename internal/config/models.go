package config

type Configuration struct {
	Default   DefaultConfig `yaml:"default"`
	Pipelines []Pipeline    `yaml:"pipelines"`
}

type DefaultConfig struct {
	Permissions  []string     `yaml:"permissions"`
	Repositories []Repository `yaml:"repositories"`
}

type Permissions struct {
	Permissions []string `yaml:"permissions"`
}

type Repository struct {
	Name        string   `yaml:"name"`
	Permissions []string `yaml:"permissions,omitempty"`
}

type Pipeline struct {
	Name         string       `yaml:"name"`
	Repositories []Repository `yaml:"repositories"`
}
