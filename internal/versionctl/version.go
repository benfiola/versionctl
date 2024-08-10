package versionctl

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// A VersionChange represents a 'type' of version bump.  A 'prerelease'
// version bump requires a prerelease token.
type VersionChange struct {
	Value           string
	PrereleaseToken string
}

// Returns an 'int' value of a version change struct - useful during comparisons
func (v VersionChange) int() int {
	if v.Value == "major" {
		return 4
	} else if v.Value == "minor" {
		return 3
	} else if v.Value == "patch" {
		return 2
	} else if v.Value == "prerelease" {
		return 1
	}
	return 0
}

// Compares the current [VersionChange] with another [VersionChange] .
// Returns < 0 if the current [VersionChange] results in a smaller version bump
// Returns 0 if the current [VersionChange] results in an equivalent version bump
// Returns > 1 if the current [VersionChange] results in a larger version bump
func (l VersionChange) Compare(r VersionChange) int {
	return cmp.Compare(l.int(), r.int())
}

// A Prerelease represents the prerelease components of a version
type Prerelease struct {
	Token string
	Count int
}

// A Version contains all the components that comprise a semantic version
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease Prerelease
	Metadata   string
}

// Bumps the current [Version] by the amount specified via the [VersionChange] and
// returns a new [Version].  If the [VersionChange] is a 'prerelease' change
// and the prerelease token does not match that of the current [Version], the
// prerelease token is changed and the prerelease count is reset.
func (v Version) Bump(c VersionChange) Version {
	// initialize new version with *only* release components
	// (metadata is always cleared)
	// (prerelease set *only* on prerelease version bump)
	nv := Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch}
	if c.Value == "major" {
		nv.Major += 1
		nv.Minor = 0
		nv.Patch = 0
	} else if c.Value == "minor" {
		nv.Minor += 1
		nv.Patch = 0
	} else if c.Value == "patch" {
		nv.Patch += 1
	} else if c.Value == "prerelease" {
		nv.Prerelease = Prerelease{Token: v.Prerelease.Token, Count: v.Prerelease.Count}
		if nv.Prerelease.Token != c.PrereleaseToken {
			// reset count if token doesn't match
			nv.Prerelease.Token = c.PrereleaseToken
			nv.Prerelease.Count = 0
		}
		nv.Prerelease.Count += 1
	}
	return nv
}

// Compares the current [Version] with another [Version].
// Returns < 0 if the current [Version] is less than the other [Version].
// Return 0 if the current [Version] is equal to the other [Version].
// Returns > 0 if the current [Version] is greater than the other [Version].
// Prerelease considered 'less than' release
// Ignores metadata
func (l Version) Compare(r Version) int {
	lvs := []int{l.Major, l.Minor, l.Patch, 0}
	if l.Prerelease == (Prerelease{}) {
		lvs[3] = 1
	}
	rvs := []int{r.Major, r.Minor, r.Patch, 0}
	if r.Prerelease == (Prerelease{}) {
		rvs[3] = 1
	}
	for i := 0; i < 4; i++ {
		d := cmp.Compare(lvs[i], rvs[i])
		if d != 0 {
			return d
		}
	}
	return 0
}

// Compares the current [Version] with another [Version] and returns
// the maximal difference between the versions by returning a [VersionChange]
// object.
func (l Version) Diff(r Version) VersionChange {
	v := "none"
	if l.Major != r.Major {
		v = "major"
	} else if l.Minor != r.Minor {
		v = "minor"
	} else if l.Patch != r.Patch {
		v = "patch"
	}
	return VersionChange{Value: v}
}

// Returns a 'release' [Version] (i.e., prerelease and metadata components removed)
// from the current [Version].
func (v Version) Release() Version {
	return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch}
}

// Returns a string representation of [Version].
// Defaults to 'semver' when format not specified, or format unrecognized.
// docker: semver, replaces '+' with '-'
// git: adds 'v' prefix to semver
// node: semver, replaces '+' with '-'
// semver: semantic version representation
func (v Version) String(f string) string {
	if f == "docker" {
		sv := v.String("semver")
		s := strings.Replace(sv, "+", "-", -1)
		return s
	} else if f == "git" {
		sv := v.String("semver")
		s := fmt.Sprintf("v%s", sv)
		return s
	} else if f == "node" {
		sv := v.String("semver")
		s := strings.Replace(sv, "+", "-", -1)
		return s
	} else if f == "semver" {
		s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
		if v.Prerelease != (Prerelease{}) {
			s = fmt.Sprintf("%s-%s.%d", s, v.Prerelease.Token, v.Prerelease.Count)
		}
		if v.Metadata != "" {
			s = fmt.Sprintf("%s+%s", s, v.Metadata)
		}
		return s
	} else {
		return v.String("semver")
	}
}

var versionRegex = regexp.MustCompile(
	"(?P<major>\\d+)" +
		"\\.(?P<minor>\\d+)" +
		"\\.(?P<patch>\\d+)" +
		"(?:-(?P<prereleaseToken>.+)\\.(?P<prereleaseCount>\\d+))?" +
		"(?:\\+(?P<metadata>.+))?")

// Creates a [Version] from a given semantic version string
func NewVersion(v string) (Version, error) {
	m := versionRegex.FindStringSubmatch(v)
	if m == nil {
		return Version{}, fmt.Errorf("invalid version string %s", v)
	}

	extractStr := func(n string) (string, error) {
		i := versionRegex.SubexpIndex(n)
		if i == -1 {
			return "", fmt.Errorf("capture group %s not found", n)
		}
		return m[i], nil
	}

	extractInt := func(n string) (int, error) {
		vs, err := extractStr(n)
		if err != nil {
			return -1, err
		}
		v, err := strconv.ParseInt(vs, 0, 0)
		if err != nil {
			return -1, fmt.Errorf("invalid %s component %w", n, err)
		}
		return int(v), nil
	}

	ma, err := extractInt("major")
	if err != nil {
		return Version{}, err
	}
	mi, err := extractInt("minor")
	if err != nil {
		return Version{}, err
	}
	p, err := extractInt("patch")
	if err != nil {
		return Version{}, err
	}
	pr := Prerelease{}
	prt, err := extractStr("prereleaseToken")
	if err != nil {
		return Version{}, err
	}
	if prt != "" {
		prc, err := extractInt("prereleaseCount")
		if err != nil {
			return Version{}, err
		}
		pr.Token = prt
		pr.Count = prc
	}
	me, err := extractStr("metadata")
	if err != nil {
		return Version{}, err
	}
	return Version{Major: ma, Minor: mi, Patch: p, Prerelease: pr, Metadata: me}, nil
}

// Writes a version string to a known file.  If the file is
// unrecognized, an error is raised.  If any part of the file
// operation fails, an error is raised.
func SetVersion(v string, f string) error {
	s, err := os.Stat(f)
	if err != nil {
		return err
	}
	if s.Name() == "pyproject.toml" {
		fd, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		d := map[string]any{}
		toml.Unmarshal(fd, &d)
		_, ok := d["project"]
		if !ok {
			d["project"] = map[string]any{}
		}
		d["project"].(map[string]any)["version"] = v
		fd, err = toml.Marshal(d)
		if err != nil {
			return err
		}
		err = os.WriteFile(f, fd, 0o644)
		if err != nil {
			return err
		}
	} else if s.Name() == "package.json" {
		fd, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		d := map[string]any{}
		json.Unmarshal(fd, &d)
		d["version"] = v
		fd, err = json.Marshal(d)
		if err != nil {
			return err
		}
		err = os.WriteFile(f, fd, 0o644)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unknown file %s", f)
	}
	return nil
}
