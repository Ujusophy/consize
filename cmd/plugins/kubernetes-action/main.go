package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/plugin/marketplace"
	"github.com/consize-oss/consize/pkg/plugins/kubernetes"
	"os"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-manifest" {
		if err := json.NewEncoder(os.Stdout).Encode(kubernetes.NewWithPatcher(nil).Manifest()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	err := marketplace.ServeOnce(context.Background(), os.Stdin, os.Stdout, func(raw json.RawMessage) (plugin.Plugin, error) {
		var cfg kubernetes.Config
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return nil, err
			}
		}
		return kubernetes.New(cfg)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
