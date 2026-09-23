package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/consize-oss/consize/internal/bootstrap"
	"github.com/consize-oss/consize/internal/store"
	"github.com/consize-oss/consize/pkg/resource"
)

func Load(path string, out any) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func LoadBootstrap(path string) (bootstrap.Config, error) {
	var cfg bootstrap.Config
	if err := Load(path, &cfg); err != nil {
		return bootstrap.Config{}, err
	}
	return cfg, nil
}

func LoadResource(path string) (resource.Resource, error) {
	var res resource.Resource
	if err := Load(path, &res); err != nil {
		return resource.Resource{}, err
	}
	return res, nil
}

func LoadRecommendation(path string) (store.Recommendation, error) {
	var rec store.Recommendation
	if err := Load(path, &rec); err != nil {
		return store.Recommendation{}, err
	}
	return rec, nil
}
