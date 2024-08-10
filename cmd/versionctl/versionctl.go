package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/benfiola/versionctl/internal/versionctl"
	"github.com/urfave/cli/v2"
)

// Configures logging for the application.  Accepts a logging level
// 'error' | 'warn' | 'info' | 'debug'
func configureLogging(ls string) error {
	if ls == "" {
		ls = "error"
	}
	var l slog.Level
	if ls == "error" {
		l = slog.LevelError
	} else if ls == "warn" {
		l = slog.LevelWarn
	} else if ls == "info" {
		l = slog.LevelInfo
	} else if ls == "debug" {
		l = slog.LevelDebug
	} else {
		return fmt.Errorf("unrecognized log level %s", ls)
	}
	slog.SetLogLoggerLevel(l)
	return nil
}

// Loads a config from the provided path.  If the path is
// a zero-value, load the default configuration embedded in the
// binary.
func loadConfig(p string) (versionctl.Config, error) {
	var b []byte
	if p != "" {
		_, err := os.Stat(p)
		if err != nil {
			return versionctl.Config{}, err
		}
		b, err = os.ReadFile(p)
		if err != nil {
			return versionctl.Config{}, err
		}
	} else {
		b = versionctl.DefaultConfig
	}

	var d versionctl.Config
	err := json.Unmarshal(b, &d)
	if err != nil {
		return versionctl.Config{}, err
	}
	return d, nil
}

func main() {
	err := (&cli.App{
		Usage: "a version management tool",
		Before: func(c *cli.Context) error {
			cfg, err := loadConfig(c.String("config"))
			if err != nil {
				return err
			}
			c.Context = context.WithValue(c.Context, "config", cfg)
			err = configureLogging(c.String("log-level"))
			if err != nil {
				return err
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "path to a configuration file",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "logging verbosity level",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "convert",
				Usage:     "convert a version into other formats",
				ArgsUsage: "[value] [format]",
				Action: func(c *cli.Context) error {
					v := c.Args().Get(0)
					f := c.Args().Get(1)
					vn, err := versionctl.NewVersion(v)
					if err != nil {
						return err
					}
					fmt.Printf("%s", vn.String(f))
					return nil
				},
			},
			{
				Name:  "current",
				Usage: "print the current version",
				Action: func(c *cli.Context) error {
					cfg, ok := c.Context.Value("config").(versionctl.Config)
					if !ok {
						return fmt.Errorf("context has invalid config")
					}
					a, err := versionctl.NewAnalyzer(cfg)
					if err != nil {
						return err
					}
					v, err := a.GetCurrentVersion()
					if err != nil {
						return err
					}
					fmt.Printf("%s", v.String(""))
					return nil
				},
			},
			{
				Name:  "next",
				Usage: "print the next version",
				Action: func(c *cli.Context) error {
					cfg, ok := c.Context.Value("config").(versionctl.Config)
					if !ok {
						return fmt.Errorf("context has invalid config")
					}
					a, err := versionctl.NewAnalyzer(cfg)
					if err != nil {
						return err
					}
					v, err := a.GetNextVersion()
					if err != nil {
						return err
					}
					fmt.Fprintf(c.App.Writer, "%s", v.String(""))
					return nil
				},
			},
			{
				Name:      "set",
				Usage:     "set version field for known files",
				ArgsUsage: "[file] [version]",
				Action: func(c *cli.Context) error {
					f := c.Args().Get(0)
					v := c.Args().Get(1)
					err := versionctl.SetVersion(f, v)
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:  "version",
				Usage: "print the tool version",
				Action: func(c *cli.Context) error {
					v := strings.TrimSpace(versionctl.VersionctlVersion)
					fmt.Fprintf(c.App.Writer, "%s", v)
					return nil
				},
			},
		},
	}).Run(os.Args)

	code := 0
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		code = 1
	}
	os.Exit(code)
}
