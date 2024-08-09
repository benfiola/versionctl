package versionctl

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/pelletier/go-toml/v2"
)

type VersionChange struct {
	Value           string
	PrereleaseToken *string
}

func (lvc VersionChange) Compare(rvc VersionChange) int {
	return cmp.Compare(lvc.Int(), rvc.Int())
}

func (vc VersionChange) Int() int {
	if vc.Value == "major" {
		return 4
	} else if vc.Value == "minor" {
		return 3
	} else if vc.Value == "patch" {
		return 2
	} else if vc.Value == "prerelease" {
		return 1
	} else {
		return 0
	}
}

type Prerelease struct {
	Token string
	Count int64
}

type Version struct {
	Major      int64
	Minor      int64
	Patch      int64
	Prerelease *Prerelease
	Metadata   *string
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

func (lv Version) Compare(rv Version) int {
	lvPrerelease := 0
	if lv.Prerelease == nil {
		lvPrerelease = 1
	}
	lvValues := [4]int64{lv.Major, lv.Minor, lv.Patch, int64(lvPrerelease)}

	rvPrerelease := 0
	if rv.Prerelease == nil {
		rvPrerelease = 1
	}
	rvValues := [4]int64{rv.Major, rv.Minor, rv.Patch, int64(rvPrerelease)}

	for i := 0; i < 4; i++ {
		diff := cmp.Compare(lvValues[i], rvValues[i])
		if diff != 0 {
			return diff
		}
	}
	return 0
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

func (v Version) Release() Version {
	return Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}
}

func (v Version) String(format string) (string, error) {
	if format == "semver" {
		value := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
		if v.Prerelease != nil {
			value = fmt.Sprintf("%s-%s.%d", value, v.Prerelease.Token, v.Prerelease.Count)
		}
		if v.Metadata != nil {
			value = fmt.Sprintf("%s+%s", value, *v.Metadata)
		}
		return value, nil
	} else {
		return "", fmt.Errorf("not implemented: %s", format)
	}
}

var versionRegex *regexp.Regexp = regexp.MustCompile(
	"(?P<major>\\d+)" +
		"\\.(?P<minor>\\d+)" +
		"\\.(?P<patch>\\d+)" +
		"(?:-(?P<prereleaseToken>.+)\\.(?P<prereleaseCount>\\d+))?" +
		"(?:\\+(?P<metadata>.+))?")

func NewVersion(value string) (Version, error) {
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
	major, err := strconv.ParseInt(matches[majorIndex], 0, 0)
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version value: %w", err)
	}

	minorIndex, err := subexpIndex("minor")
	if err != nil {
		return Version{}, err
	}
	minor, err := strconv.ParseInt(matches[minorIndex], 0, 0)
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version value: %w", err)
	}

	patchIndex, err := subexpIndex("patch")
	if err != nil {
		return Version{}, err
	}
	patch, err := strconv.ParseInt(matches[patchIndex], 0, 0)
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
		prereleaseCount, err := strconv.ParseInt(matches[prereleaseCountIndex], 0, 0)
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

func SetVersion(filePath string, version string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("could not open version file %s", err)
	}

	if stat.Name() == "pyproject.toml" {
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		var data map[string]any
		err = toml.Unmarshal(fileBytes, &data)
		if err != nil {
			return err
		}
		projectMap, ok := data["project"].(map[string]any)
		if !ok {
			return fmt.Errorf("malformed pyproject.toml")
		}
		projectMap["version"] = version
		fileBytes, err = toml.Marshal(data)
		if err != nil {
			return err
		}
		err = os.WriteFile(filePath, fileBytes, 0644)
		if err != nil {
			return err
		}
	} else if stat.Name() == "package.json" {
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		var data map[string]any
		err = json.Unmarshal(fileBytes, &data)
		if err != nil {
			return err
		}
		data["version"] = version
		fileBytes, err = json.Marshal(data)
		if err != nil {
			return err
		}
		err = os.WriteFile(filePath, fileBytes, 0644)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported file: %s", filePath)
	}
	return nil
}
