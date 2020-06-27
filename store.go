package main

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type storage struct {
	LastUID uint32
}

func loadStorage() (*storage, error) {
	var out = &storage{}

	if _, err := os.Stat(cfg.StorageFile); os.IsNotExist(err) {
		return out, nil
	}

	f, err := os.Open(cfg.StorageFile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open storage file")
	}
	defer f.Close()

	return out, errors.Wrap(yaml.NewDecoder(f).Decode(out), "Unable to decode storage file")
}

func (s storage) saveStorage() error {
	if err := os.MkdirAll(path.Dir(cfg.StorageFile), 0700); err != nil {
		return errors.Wrap(err, "Unable to ensure directory for storage file")
	}

	f, err := os.Create(cfg.StorageFile)
	if err != nil {
		return errors.Wrap(err, "Unable to create storage file")
	}
	defer f.Close()

	return errors.Wrap(yaml.NewEncoder(f).Encode(s), "Unable to encode storage file")
}
