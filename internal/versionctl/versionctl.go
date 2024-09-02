package versionctl

import (
	"io"
	"log/slog"
)

// A Config represents the entire configuration object used to configure versionctl behavior.
type Config struct {
	BreakingChangeTags []string          `json:"breakingChangeTags"`
	Parser             string            `json:"parser"`
	Rules              []Rule            `json:"rules"`
	Tags               map[string]string `json:"tags"`
}

// Options provided to the entry point [New].
type Opts struct {
	Config *Config
	Logger *slog.Logger
}

// Entry point of the application.
// Initializes subcomponents, returns [Analyzer].
// Returns an error if any subcomponents fail to initialize.
func New(o *Opts) (*Analyzer, error) {
	l := o.Logger
	if l == nil {
		l = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	p, err := NewParser(o.Config.Parser, &ParserOpts{
		BreakingChangeTags: o.Config.BreakingChangeTags,
		Logger:             l.With("name", "parser"),
		Tags:               o.Config.Tags,
	})
	if err != nil {
		return nil, err
	}
	g, err := NewGit(&GitOpts{
		Logger: l.With("name", "git"),
	})
	if err != nil {
		return nil, err
	}
	a, err := NewAnalyzer(&AnalyzerOpts{
		Git:    g,
		Logger: l.With("name", "analyzer"),
		Parser: p,
		Rules:  o.Config.Rules,
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}
