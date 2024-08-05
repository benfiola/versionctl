package version

import (
	"fmt"
	"regexp"
	"strconv"
)

type VersionChange struct {
	Value           string
	PrereleaseToken *string
}

func (change VersionChange) ToInt() int {
	if change.Value == "major" {
		return 4
	} else if change.Value == "minor" {
		return 3
	} else if change.Value == "patch" {
		return 2
	} else if change.Value == "prerelease" {
		return 1
	} else {
		return 0
	}
}

type Prerelease struct {
	Token string
	Count uint64
}

type Version struct {
	Major      uint64
	Minor      uint64
	Patch      uint64
	Prerelease *Prerelease
	Metadata   *string
}

func (left Version) Compare(right Version) int8 {

}

func (version Version) Bump(change VersionChange) (Version, error) {
	major := version.Major
	minor := version.Minor
	patch := version.Patch
	var prerelease *Prerelease

	if change.Value == "major" {
		major += 1
		minor = 0
		patch = 0
	} else if change.Value == "minor" {
		minor += 1
		patch = 0
	} else if change.Value == "patch" {
		patch += 1
	} else if change.Value == "prerelease" {
		if change.PrereleaseToken == nil {
			return Version{}, fmt.Errorf("prerelease token undefined")
		}
		prerelease = &Prerelease{
			Token: *change.PrereleaseToken,
			Count: 0,
		}
		if version.Prerelease != nil && version.Prerelease.Token == prerelease.Token {
			prerelease.Count = version.Prerelease.Count
		}
		prerelease.Count += 1
	} else {
		return Version{}, fmt.Errorf("not implemented: %#v", change)
	}

	return Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, nil
}

func (left Version) Diff(right Version) VersionChange {
	value := "none"
	if left.Major != right.Major {
		value = "major"
	} else if left.Minor != right.Minor {
		value = "minor"
	} else if left.Patch != right.Patch {
		value = "patch"
	}

	var prereleaseToken *string
	return VersionChange{
		Value:           value,
		PrereleaseToken: prereleaseToken,
	}
}

var versionRegex *regexp.Regexp = regexp.MustCompile(
	"(?P<major>\\d+)" +
		"\\.(?P<minor>\\d+)" +
		"\\.(?P<patch>\\d+)" +
		"(?:-(?P<prereleaseToken>.+)\\.(?P<prereleaseCount>\\d+))?" +
		"(?:\\+(?P<metadata>.+))?")

func New(value string) (Version, error) {
	subexpIndex := func(name string) (int, error) {
		index := versionRegex.SubexpIndex(name)
		if index == -1 {
			return -1, fmt.Errorf("named group %s does not exist", name)
		}
		return index, nil
	}

	matches := versionRegex.FindStringSubmatch(value)
	if matches == nil {
		return Version{}, fmt.Errorf("invalid semver version string: %s", value)
	}

	majorIndex, err := subexpIndex("major")
	if err != nil {
		return Version{}, err
	}
	major, err := strconv.ParseUint(matches[majorIndex], 10, 64)
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version value: %w", err)
	}

	minorIndex, err := subexpIndex("minor")
	if err != nil {
		return Version{}, err
	}
	minor, err := strconv.ParseUint(matches[minorIndex], 10, 64)
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version value: %w", err)
	}

	patchIndex, err := subexpIndex("patch")
	if err != nil {
		return Version{}, err
	}
	patch, err := strconv.ParseUint(matches[patchIndex], 10, 64)
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version value: %w", err)
	}

	prereleaseTokenIndex, err := subexpIndex("prereleaseToken")
	if err != nil {
		return Version{}, err
	}
	prereleaseCountIndex, err := subexpIndex("prereleaseCount")
	if err != nil {
		return Version{}, err
	}

	var prerelease *Prerelease
	if matches[prereleaseCountIndex] != "" && matches[prereleaseTokenIndex] != "" {
		prereleaseToken := matches[prereleaseTokenIndex]
		prereleaseCount, err := strconv.ParseUint(matches[prereleaseCountIndex], 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("invalid prerelease value: %w", err)
		}
		prerelease = &Prerelease{
			Token: prereleaseToken,
			Count: prereleaseCount,
		}
	} else if matches[prereleaseCountIndex] != "" || matches[prereleaseTokenIndex] != "" {
		return Version{}, fmt.Errorf("invalid prerelease value: token or count not defined")
	}

	metadataIndex, err := subexpIndex("metadata")
	if err != nil {
		return Version{}, err
	}
	var metadata *string
	if matches[metadataIndex] != "" {
		temp := matches[metadataIndex]
		metadata = &temp
	}

	return Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Metadata:   metadata,
	}, nil
}

func (version Version) String(format string) (string, error) {
	if format == "semver" {
		value := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)
		if version.Prerelease != nil {
			value = fmt.Sprintf("%s-%s.%d", value, version.Prerelease.Token, version.Prerelease.Count)
		}
		if version.Metadata != nil {
			value = fmt.Sprintf("%s+%s", value, *version.Metadata)
		}
		return value, nil
	} else {
		return "", fmt.Errorf("not implemented: %s", format)
	}
}
