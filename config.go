package main

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type config struct {
	Handlers []mailHandler `yaml:"handlers"`
}

func loadConfig() (*config, error) {
	var out = &config{}

	f, err := os.Open(cfg.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to open config file")
	}
	defer f.Close()

	return out, errors.Wrap(yaml.NewDecoder(f).Decode(out), "Unable to decode config")
}
