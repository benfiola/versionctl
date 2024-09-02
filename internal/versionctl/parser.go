package versionctl

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// A parser parses commit messages to determine what type of version bump is mandated by the commit.
// To support multiple types of parsers, a parser is presented as a simple parsing interface
type Parser interface {
	Parse(message string) VersionChange
}

// Default options accepted by all parser implementations
type ParserOpts struct {
	BreakingChangeTags []string // tags in the commit body that will result in a 'major' version bump
	Logger             *slog.Logger
	Tags               map[string]string // tags in the commit header that map to version bump values
}

// A 'default' parser
type defaultParser struct {
	breakingChangeTags []string
	logger             *slog.Logger
	tags               map[string]string
}

// Creates a new [Parser] from the given parser type and options.
// Returns an error if the parser type is invalid
func NewParser(k string, o *ParserOpts) (Parser, error) {
	l := o.Logger
	if l == nil {
		l = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if k == "" {
		k = "default"
	}
	switch k {
	case "default":
		return &defaultParser{
			breakingChangeTags: o.BreakingChangeTags,
			logger:             l,
			tags:               o.Tags,
		}, nil
	default:
		return nil, fmt.Errorf("invalid parser type %s", k)
	}
}

// Parses the given message.  Expects the commit message to contain at least one line (a 'header') and optional, additional lines (a 'body').
// Expects the header to start with a tag specified in [defaultParser.tags].
// If neither expectaions are met, returns a 'none' version change.
// If a line from the body starts with a tag specified in [defaultParser.breakingChangeTags] - will return a major version change.
func (p defaultParser) Parse(message string) VersionChange {
	ls := strings.Split(message, "\n")

	h := ls[0]
	v := ""
	for t, tv := range p.tags {
		if !strings.HasPrefix(h, t) {
			continue
		}
		v = tv
		break
	}
	if v == "" {
		return VersionChange{Value: "none"}
	}

	b := []string{}
	if len(ls) > 1 {
		b = ls[1:]
	}
	for _, l := range b {
		if v == "major" {
			break
		}
		for _, bct := range p.breakingChangeTags {
			if !strings.HasPrefix(l, bct) {
				continue
			}
			v = "major"
			break
		}
	}
	return VersionChange{Value: v}
}
