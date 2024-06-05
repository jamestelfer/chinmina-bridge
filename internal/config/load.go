package config

import (
	"bytes"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadFromFile(path string) (*Configuration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfiguration(data)
}

func LoadAndEncodeConfig(filePath string) (string, error) {
	config, err := LoadFromFile(filePath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	err = enc.Encode(config)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
