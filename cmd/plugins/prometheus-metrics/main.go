package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/plugin/marketplace"
	"github.com/consize-oss/consize/pkg/plugins/prometheus"
	"os"
	"time"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-manifest" {
		if err := json.NewEncoder(os.Stdout).Encode(prometheus.NewWithClient(nil, prometheus.Config{}).Manifest()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	err := marketplace.ServeOnce(context.Background(), os.Stdin, os.Stdout, func(raw json.RawMessage) (plugin.Plugin, error) {
		var cfg struct {
			BaseURL string            `json:"base_url"`
			Window  string            `json:"window"`
			Step    string            `json:"step"`
			Queries map[string]string `json:"queries"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, err
		}
		var window, step time.Duration
		var err error
		if cfg.Window != "" {
			window, err = time.ParseDuration(cfg.Window)
			if err != nil {
				return nil, err
			}
		}
		if cfg.Step != "" {
			step, err = time.ParseDuration(cfg.Step)
			if err != nil {
				return nil, err
			}
		}
		return prometheus.New(prometheus.Config{BaseURL: cfg.BaseURL, Window: window, Step: step, Queries: cfg.Queries})
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
