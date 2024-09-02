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

type ContextOpts struct{}

// Creates a root logger for the application.
// Accepts a logging level 'error' | 'warn' | 'info' | 'debug'
// Returns an error if the logging level is invalid
func createLogger(lls string) (*slog.Logger, error) {
	if lls == "" {
		lls = "error"
	}
	var ll slog.Level
	switch lls {
	case "error":
		ll = slog.LevelError
	case "warn":
		ll = slog.LevelWarn
	case "info":
		ll = slog.LevelInfo
	case "debug":
		ll = slog.LevelDebug
	default:
		return nil, fmt.Errorf("invalid log level %s", lls)
	}
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: ll,
	}))
	return l, nil
}

// Loads a config from the provided path.  If the path is
// a zero-value, load the default configuration embedded in the
// binary.
func loadConfig(p string) (*versionctl.Config, error) {
	var b []byte
	var err error
	if p != "" {
		b, err = os.ReadFile(p)
		if err != nil {
			return nil, err
		}
	} else {
		b = versionctl.DefaultConfig
	}

	cfg := &versionctl.Config{}
	err = json.Unmarshal(b, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func main() {
	err := (&cli.App{
		Usage: "a version management tool",
		Before: func(c *cli.Context) error {
			cfg, err := loadConfig(c.String("config"))
			if err != nil {
				return err
			}
			l, err := createLogger(c.String("log-level"))
			if err != nil {
				return err
			}
			o := versionctl.Opts{
				Config: cfg,
				Logger: l,
			}
			c.Context = context.WithValue(c.Context, ContextOpts{}, &o)
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
					fmt.Fprintf(c.App.Writer, "%s", vn.String(f))
					return nil
				},
			},
			{
				Name:  "current",
				Usage: "print the current version",
				Action: func(c *cli.Context) error {
					o, ok := c.Context.Value(ContextOpts{}).(*versionctl.Opts)
					if !ok {
						return fmt.Errorf("context has invalid opts")
					}
					a, err := versionctl.New(o)
					if err != nil {
						return err
					}
					v, err := a.GetCurrentVersion()
					if err != nil {
						return err
					}
					fmt.Fprintf(c.App.Writer, "%s", v.String(""))
					return nil
				},
			},
			{
				Name:  "next",
				Usage: "print the next version",
				Action: func(c *cli.Context) error {
					o, ok := c.Context.Value(ContextOpts{}).(*versionctl.Opts)
					if !ok {
						return fmt.Errorf("context has invalid opts")
					}
					a, err := versionctl.New(o)
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
