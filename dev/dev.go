package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/benfiola/versionctl/internal/versionctl"
)

func inner() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	wd = path.Dir(wd)
	err = os.Chdir(wd)
	if err != nil {
		return err
	}

	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	cfg := &versionctl.Config{}
	err = json.Unmarshal(versionctl.DefaultConfig, cfg)
	if err != nil {
		return err
	}
	a, err := versionctl.New(&versionctl.Opts{
		Config: cfg,
		Logger: l,
	})
	if err != nil {
		return err
	}

	nv, err := a.GetNextVersion()
	if err != nil {
		return err
	}
	fmt.Printf("next version: %s", nv.String("semver"))
	return nil
}

func main() {
	err := inner()
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit((1))
	}
}
