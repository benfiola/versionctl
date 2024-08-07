package versionctl

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type VersionRule struct {
	Branch          string
	BuildMetadata   string
	PrereleaseToken string
}

type Config struct {
	BreakingChangeTags []string
	Rules              []VersionRule
	Tags               map[string]string
}

type Analyzer struct {
	Git    Git
	Parser Parser
	Rules  []VersionRule
}

func NewFromConfig(c *Config) (Analyzer, error) {
	g, err := NewGit("")
	if err != nil {
		return Analyzer{}, err
	}
	p := Parser{BreakingChangeTags: c.BreakingChangeTags, Tags: c.Tags}
	a := Analyzer{Git: g, Parser: p, Rules: c.Rules}
	return a, nil
}

func (a Analyzer) parseVersions(tags []string) []Version {
	versions := []Version{}
	for _, tag := range tags {
		versionStr, found := strings.CutPrefix(tag, "v")
		if !found {
			continue
		}
		version, err := NewVersion(versionStr)
		if err != nil {
			continue
		}
		versions = append(versions, version)
	}
	return versions
}

func (a Analyzer) getRepoData() (Version, error) {
	tags, err := a.Git.ListTags()
	if err != nil {
		return Version{}, err
	}

	versions := a.parseVersions(tags)

	if len(versions) == 0 {
		version, err := NewVersion("0.0.0")
		if err != nil {
			return Version{}, err
		}
		versions = append(versions, version)
	}

	slices.SortFunc(versions, CompareVersion)
	slices.Reverse(versions)

	return versions[0], nil
}

func (a Analyzer) getAncestorData() (Version, VersionChange, error) {
	change := VersionChange{Value: "none"}
	var release *Version
	err := a.Git.IterCommits("", func(c Commit) error {
		versions := a.parseVersions(c.Tags)
		for _, version := range versions {
			if version.Prerelease == nil {
				release = &version
				break
			}
		}
		if release != nil {
			return &StopIter{}
		}
		cChange := a.Parser.Parse(c.Message)
		if CompareVersionChange(change, cChange) < 0 {
			change = cChange
		}
		return nil
	})
	if err != nil {
		return Version{}, VersionChange{}, err
	}
	if release == nil {
		r, err := NewVersion("0.0.0")
		if err != nil {
			return Version{}, VersionChange{}, err
		}
		release = &r
	}
	return *release, change, nil
}

func (a Analyzer) matchRule(branch string) (VersionRule, error) {
	for _, rule := range a.Rules {
		re, err := regexp.Compile(rule.Branch)
		if err != nil {
			return VersionRule{}, err
		}
		if re.Match([]byte(branch)) {
			return rule, nil
		}
	}
	return VersionRule{}, fmt.Errorf("no rule found: %s", branch)
}

func (a Analyzer) GetCurrentVersion() (Version, error) {
	version, err := a.getRepoData()
	if err != nil {
		return Version{}, err
	}
	return version, nil
}

func (a Analyzer) GetNextVersion() (Version, error) {
	branch, err := a.Git.GetCurrentBranch()
	if err != nil {
		return Version{}, err
	}

	rule, err := a.matchRule(branch)
	if err != nil {
		return Version{}, err
	}

	repoVersion, err := a.getRepoData()
	if err != nil {
		return Version{}, err
	}

	ancestorRelease, change, err := a.getAncestorData()
	if err != nil {
		return Version{}, err
	}
	if change.Value == "none" {
		return Version{}, fmt.Errorf("version unchanged")
	}

	drift := repoVersion.Diff(ancestorRelease)
	var version Version
	if rule.PrereleaseToken != "" {
		if CompareVersionChange(drift, change) < 0 {
			version, err = repoVersion.Bump(change)
			if err != nil {
				return Version{}, err
			}
		} else {
			version = repoVersion
		}
		version, err = version.Bump(VersionChange{Value: "prerelease", PrereleaseToken: &rule.PrereleaseToken})
		if err != nil {
			return Version{}, err
		}
	} else {
		if repoVersion.Prerelease == nil {
			version, err = repoVersion.Bump(change)
			if err != nil {
				return Version{}, err
			}
		} else {
			if CompareVersionChange(drift, change) < 0 {
				version, err = repoVersion.Bump(change)
				if err != nil {
					return Version{}, err
				}
			} else {
				version = repoVersion.Release()
			}
		}
	}

	return version, nil
}
