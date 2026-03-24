package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Terminal string `yaml:"terminal"`
}

var Current Config

func Load() error {
	// Try multiple locations
	locations := []string{"internal/config.yml", "config.yml"}
	var data []byte
	var err error

	for _, loc := range locations {
		data, err = os.ReadFile(loc)
		if err == nil {
			break
		}
	}

	if err != nil {
		// If no config found, default to auto
		Current.Terminal = "auto"
		return nil
	}

	err = yaml.Unmarshal(data, &Current)
	if err != nil {
		return err
	}

	if Current.Terminal == "" {
		Current.Terminal = "auto"
	}

	return nil
}
