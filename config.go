package main

import (
	"os"

	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
)

type config struct {
	Handlers []mailHandler `yaml:"handlers"`
}

func loadConfig() (*config, error) {
	out := &config{}

	f, err := os.Open(cfg.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to open config file")
	}
	defer f.Close() //nolint:errcheck

	return out, errors.Wrap(yaml.NewDecoder(f).Decode(out), "Unable to decode config")
}
