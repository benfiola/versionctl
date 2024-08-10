package versionctl

import (
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
)

// Represents a rule match.
type RuleMatch struct {
	Matched bool
	Data    map[string]string
	Rule    Rule
}

// A Rule matches a branch name with specific version change behavior
type Rule struct {
	Branch          string
	PrereleaseToken string
	Metadata        string
}

// Matches a branch name to a given [Rule].
// Returns zero-value if no match.
func (r Rule) Match(b string) (RuleMatch, error) {
	bre, err := regexp.Compile(r.Branch)
	if err != nil {
		return RuleMatch{}, err
	}
	m := bre.FindStringSubmatch(b)
	if m == nil {
		return RuleMatch{}, err
	}
	d := map[string]string{}
	for _, n := range bre.SubexpNames()[1:] {
		i := bre.SubexpIndex(n)
		d[n] = m[i]
	}
	return RuleMatch{Matched: true, Data: d, Rule: r}, nil
}

// A Config represents the entire configuration object used to configure
// versionctl behavior.
type Config struct {
	BreakingChangeTags []string
	Rules              []Rule
	Tags               map[string]string
}

// An Analyzer uses local repository data alongside configured rules
// to manage software versions
type Analyzer struct {
	Git    Git
	Parser Parser
	Rules  []Rule
}

// Parses a list of tags into [Version] structs, sorts them and returns them.
// Tags that aren't prefixed with 'v' (e.g, v1.0.0) are discarded.
// Once stripped of the 'v' prefix, tags that aren't version parseable are discarded.
func (a Analyzer) getSortedVersionsFromTags(ts []string) []Version {
	vs := []Version{}
	for _, t := range ts {
		if !strings.HasPrefix(t, "v") {
			// ignore tags without 'v' prefix
			continue
		}
		// remove 'v' prefix
		t = t[1:]
		//collect parseable versions
		v, err := NewVersion(t)
		if err != nil {
			continue
		}
		vs = append(vs, v)
	}
	// sort and reverse collected versions
	slices.SortFunc(vs, func(a Version, b Version) int {
		return a.Compare(b)
	})
	slices.Reverse(vs)
	return vs
}

// Represents repo-wide information used to inform version bump behavior
type repoData struct {
	Version Version // Highest version in entire repositroy
}

// Analyzes local repository and returns a [repoData].
func (a Analyzer) getRepoData() (repoData, error) {
	v := Version{}
	ts, err := a.Git.ListTags()
	if err != nil {
		return repoData{}, err
	}
	vs := a.getSortedVersionsFromTags(ts)
	if len(vs) > 0 {
		v = vs[0]
	}
	return repoData{Version: v}, nil
}

// Obtains commit ancestor information used to inform version bump behavior
type ancestorData struct {
	Version       Version       // The highest non-prerelease version in the commit ancestry
	VersionChange VersionChange // The largest change between the head and the highest non-prerelease version in the commit ancestry
}

// Analyzes a commit's ancestry (starting from HEAD) and creates an [ancestorData].
func (a Analyzer) getAncestorData() (ancestorData, error) {
	v := Version{}
	vc := VersionChange{Value: "none"}

	err := a.Git.IterCommits("", func(c GitCommit) error {
		// collect *only* release versions attached to current commit
		cvs := []Version{}
		for _, cv := range a.getSortedVersionsFromTags(c.Tags) {
			if cv.Prerelease != (Prerelease{}) {
				continue
			}
			cvs = append(cvs, cv)
		}

		// only process commit if commit not part of release
		if len(cvs) == 0 {
			cvc := a.Parser.parse(c.Message)
			slog.Debug(fmt.Sprintf("commit: %s (change: %s)", c.Hash, cvc.Value))
			if vc.Compare(cvc) < 0 {
				vc = cvc
			}
			return nil
		}

		// stop iteration - commit part of release
		v = cvs[0]
		slog.Debug(fmt.Sprintf("commit: %s (release: %s)", c.Hash, v.String("")))
		return &StopIter{}
	})
	if err != nil {
		return ancestorData{}, nil
	}
	return ancestorData{Version: v, VersionChange: vc}, nil
}

// Matches a branch name to a [Rule].
// Returns an error if no [Rule] could be found.
func (a Analyzer) findRule(bn string) (RuleMatch, error) {
	for _, r := range a.Rules {
		m, err := r.Match(bn)
		if err != nil {
			return RuleMatch{}, err
		}
		if !m.Matched {
			continue
		}
		return m, nil
	}
	return RuleMatch{}, fmt.Errorf("no rule found for %s", bn)
}

// Gets the current [Version] for the local repository.
func (a Analyzer) GetCurrentVersion() (Version, error) {
	rd, err := a.getRepoData()
	if err != nil {
		return Version{}, err
	}
	return rd.Version, nil
}

// Used to replace invalid characters in prerelease tokens or metadata
// with valid characters (probably a '-').
var nonAlphaNumericRegex = regexp.MustCompile("[^a-zA-Z0-9]+")

// Gets the next [Version] for the local repository.
func (a Analyzer) GetNextVersion() (Version, error) {
	b, err := a.Git.GetCurrentBranch()
	if err != nil {
		return Version{}, err
	}
	slog.Info(fmt.Sprintf("branch: %s", b))
	rm, err := a.findRule(b)
	r := rm.Rule
	if err != nil {
		return Version{}, err
	}
	slog.Info(fmt.Sprintf("rule: %s", r.Branch))
	rd, err := a.getRepoData()
	if err != nil {
		return Version{}, err
	}

	slog.Info(fmt.Sprintf("repo version: %s", rd.Version.String("")))
	ad, err := a.getAncestorData()
	if err != nil {
		return Version{}, err
	}
	if ad.VersionChange.Value == "none" {
		return Version{}, fmt.Errorf("version unchanged")
	}
	slog.Info(fmt.Sprintf("ancestor version: %s", ad.Version.String("")))
	slog.Info(fmt.Sprintf("ancestor change: %s", ad.VersionChange.Value))

	d := ad.Version.Diff(rd.Version)
	slog.Info(fmt.Sprintf("repo + ancestor version diff: %s", d.Value))

	var version Version
	if r.PrereleaseToken != "" {
		// rule is prerelease
		if d.Compare(ad.VersionChange) < 0 {
			// ancestor <-> repo diff is less than largest change
			// bump version
			version = rd.Version.Bump(ad.VersionChange)
		} else {
			// ancestor <-> repo diff is bigger than largest change
			// no bump needed
			version = rd.Version
		}
		// bump prerelease version
		pt := a.injectData(rm.Data, r.PrereleaseToken)
		pt = nonAlphaNumericRegex.ReplaceAllString(pt, "-")
		version = version.Bump(VersionChange{Value: "prerelease", PrereleaseToken: pt})
	} else {
		// rule is not prerelease
		if rd.Version.Prerelease == (Prerelease{}) {
			// repo version is not prerelease
			// bump version
			version = rd.Version.Bump(ad.VersionChange)
		} else {
			// repo version is prerelease
			if d.Compare(ad.VersionChange) < 0 {
				// ancestor <-> repo diff bigger than largest change
				// bump version
				version = rd.Version.Bump(ad.VersionChange)
			} else {
				// ancestor <-> repo diff less than largest change
				// only strip prerelease data
				version = rd.Version.Release()
			}
		}
	}
	if r.Metadata != "" {
		// add metadata if configured
		md := a.injectData(rm.Data, r.Metadata)
		md = nonAlphaNumericRegex.ReplaceAllString(md, "-")
		version.Metadata = md
	}
	return version, nil
}

// Given a map of values, replace template fields in string
// (format: '{<key>}') with respective map values.
// Returns a string with values replaced
func (a Analyzer) injectData(d map[string]string, v string) string {
	for key, value := range d {
		s := fmt.Sprintf("{%s}", key)
		v = strings.ReplaceAll(v, s, value)
	}
	return v
}

// Creates a new [Analyzer] from the provided [Config].
func NewAnalyzer(c Config) (Analyzer, error) {
	p := Parser{BreakingChangeTags: c.BreakingChangeTags, Tags: c.Tags}
	g, err := NewGit("")
	if err != nil {
		return Analyzer{}, err
	}
	a := Analyzer{Git: g, Parser: p, Rules: c.Rules}
	return a, nil
}
