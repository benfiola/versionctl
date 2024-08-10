package versionctl

import "strings"

// A parser parses commit messages to determine what type of version
// bump is mandated by the commit
type Parser struct {
	BreakingChangeTags []string          // tags in the commit body that will result in a 'major' version bump
	Tags               map[string]string // tags in the commit header that map to version bump values
}

// Parses the given message.  Expects the commit message to contain at least
// one line (a 'header') and optional, additional lines (a 'body').  Expects
// the header to start with a tag specified in [Parser.Tags].  If neither expectaions
// are met, returns a 'none' version change.  If a line from the body starts with
// a tag specified in [Parser.BreakingChangeTags] - will return a major version change.
func (p Parser) parse(message string) VersionChange {
	ls := strings.Split(message, "\n")

	h := ls[0]
	v := ""
	for t, tv := range p.Tags {
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
		for _, bct := range p.BreakingChangeTags {
			if !strings.HasPrefix(l, bct) {
				continue
			}
			v = "major"
			break
		}
	}
	return VersionChange{Value: v}
}
