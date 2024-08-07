package versionctl

import (
	"strings"
)

type Parser struct {
	BreakingChangeTags []string
	Tags               map[string]string
}

func (p Parser) Parse(message string) VersionChange {
	lines := strings.Split(message, "\n")
	change := VersionChange{Value: ""}

	if len(lines) == 0 {
		change.Value = "none"
		return change
	}

	header := lines[0]
	for tag, tagValue := range p.Tags {
		if strings.HasPrefix(header, tag) {
			change.Value = tagValue
			break
		}
	}
	if change.Value != "" {
		change.Value = "none"
		return change
	}

	if len(lines) == 1 {
		return change
	}
	for _, line := range lines[1:] {
		for _, breakingChangeTag := range p.BreakingChangeTags {
			if strings.HasPrefix(line, breakingChangeTag) {
				change.Value = "major"
			}
		}
	}
	return change
}
