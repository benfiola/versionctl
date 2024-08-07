package versionctl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestBump(t *testing.T) {
	createVersion := func() Version {
		metadata := "metadata"
		return Version{
			Major: 1,
			Minor: 2,
			Patch: 3,
			Prerelease: &Prerelease{
				Token: "rc",
				Count: 1,
			},
			Metadata: &metadata,
		}
	}

	t.Run("major", func(t *testing.T) {
		version := createVersion()
		bumped, error := version.Bump(VersionChange{Value: "major"})
		if !(error == nil && bumped.Major == 2 && bumped.Minor == 0 && bumped.Patch == 0 && bumped.Prerelease == nil && bumped.Metadata == nil) {
			t.Errorf("bump invalid: %#v", version)
		}
	})

	t.Run("minor", func(t *testing.T) {
		version := createVersion()
		bumped, error := version.Bump(VersionChange{Value: "minor"})
		if !(error == nil && bumped.Major == 1 && bumped.Minor == 3 && bumped.Patch == 0 && bumped.Prerelease == nil && bumped.Metadata == nil) {
			t.Errorf("bump invalid: %#v", version)
		}
	})

	t.Run("patch", func(t *testing.T) {
		version := createVersion()
		bumped, error := version.Bump(VersionChange{Value: "patch"})
		if !(error == nil && bumped.Major == 1 && bumped.Minor == 2 && bumped.Patch == 4 && bumped.Prerelease == nil && bumped.Metadata == nil) {
			t.Errorf("bump invalid: %#v", bumped)
		}
	})

	t.Run("prerelease with same token", func(t *testing.T) {
		version := createVersion()
		prereleaseToken := "rc"
		bumped, error := version.Bump(VersionChange{Value: "prerelease", PrereleaseToken: &prereleaseToken})
		if !(error == nil && bumped.Major == 1 && bumped.Minor == 2 && bumped.Patch == 3 && bumped.Prerelease.Token == "rc" && bumped.Prerelease.Count == 2 && bumped.Metadata == nil) {
			t.Errorf("bump invalid: %#v", bumped)
		}
	})

	t.Run("prerelease with different token", func(t *testing.T) {
		version := createVersion()
		prereleaseToken := "other"
		bumped, error := version.Bump(VersionChange{Value: "prerelease", PrereleaseToken: &prereleaseToken})
		if !(error == nil && bumped.Major == 1 && bumped.Minor == 2 && bumped.Patch == 3 && bumped.Prerelease.Token == "other" && bumped.Prerelease.Count == 1 && bumped.Metadata == nil) {
			t.Errorf("bump invalid: %#v", bumped)
		}
	})

	t.Run("none", func(t *testing.T) {
		version := createVersion()
		_, error := version.Bump(VersionChange{Value: "none"})
		if !(error != nil) {
			t.Errorf("bump did not fail")
		}
	})
}

func TestDiff(t *testing.T) {
	t.Run("major", func(t *testing.T) {
		left := Version{
			Major: 1,
		}
		right := Version{}
		diff := left.Diff(right)
		if diff.Value != "major" {
			t.Errorf("diff invalid: %#v", diff)
		}
	})

	t.Run("minor", func(t *testing.T) {
		left := Version{
			Minor: 1,
		}
		right := Version{}
		diff := left.Diff(right)
		if diff.Value != "minor" {
			t.Errorf("diff invalid: %#v", diff)
		}
	})

	t.Run("patch", func(t *testing.T) {
		left := Version{
			Patch: 1,
		}
		right := Version{}
		diff := left.Diff(right)
		if diff.Value != "patch" {
			t.Errorf("diff invalid: %#v", diff)
		}
	})

	t.Run("none", func(t *testing.T) {
		left := Version{}
		right := Version{}
		diff := left.Diff(right)
		if diff.Value != "none" {
			t.Errorf("diff invalid: %#v", diff)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("release", func(t *testing.T) {
		version, err := NewVersion("1.2.3")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease == nil && version.Metadata == nil) {
			t.Errorf("version invalid: %#v", version)
		}
	})

	t.Run("prerelease", func(t *testing.T) {
		version, err := NewVersion("1.2.3-rc.1")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease.Token == "rc" && version.Prerelease.Count == 1 && version.Metadata == nil) {
			t.Errorf("version invalid: %#v", version)
		}
	})

	t.Run("metadata", func(t *testing.T) {
		version, err := NewVersion("1.2.3+metadata")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease == nil && *version.Metadata == "metadata") {
			t.Errorf("version invalid: %#v", version)
		}
	})

	t.Run("prerelease and metadata", func(t *testing.T) {
		version, err := NewVersion("1.2.3-rc.1+metadata")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease.Token == "rc" && version.Prerelease.Count == 1 && *version.Metadata == "metadata") {
			t.Errorf("version invalid: %#v", version)
		}
	})
}

func TestCompareVersion(t *testing.T) {
	t.Run("major version greater than", func(t *testing.T) {
		left := Version{Major: 1}
		right := Version{}
		if !(CompareVersion(left, right) > 0) {
			t.Errorf("compare result invalid")
		}
	})

	t.Run("minor version greater than", func(t *testing.T) {
		left := Version{Minor: 1}
		right := Version{}
		if !(CompareVersion(left, right) > 0) {
			t.Errorf("compare result invalid")
		}
	})

	t.Run("patch version greater than", func(t *testing.T) {
		left := Version{Patch: 1}
		right := Version{}
		if !(CompareVersion(left, right) > 0) {
			t.Errorf("compare result invalid")
		}
	})

	t.Run("equal", func(t *testing.T) {
		left := Version{}
		right := Version{}
		if !(CompareVersion(left, right) == 0) {
			t.Errorf("comparse result invalid")
		}
	})

	t.Run("prerelease less than release", func(t *testing.T) {
		left := Version{}
		right := Version{Prerelease: &Prerelease{Token: "a", Count: 1}}
		if !(CompareVersion(left, right) > 0) {
			t.Errorf("compare result invalid")
		}
	})
}

func TestCompareVersionChange(t *testing.T) {
	t.Run("major > none", func(t *testing.T) {
		left := VersionChange{Value: "major"}
		right := VersionChange{Value: "none"}
		if !(CompareVersionChange(left, right) > 0) {
			t.Errorf("compare result invalid")
		}
	})

	t.Run("minor > none", func(t *testing.T) {
		left := VersionChange{Value: "minor"}
		right := VersionChange{Value: "none"}
		if !(CompareVersionChange(left, right) > 0) {
			t.Errorf("compare result invalid")
		}
	})

	t.Run("patch > none", func(t *testing.T) {
		left := VersionChange{Value: "patch"}
		right := VersionChange{Value: "none"}
		if !(CompareVersionChange(left, right) > 0) {
			t.Errorf("compare result invalid")
		}
	})

	t.Run("none == none", func(t *testing.T) {
		left := VersionChange{Value: "none"}
		right := VersionChange{Value: "none"}
		if !(CompareVersionChange(left, right) == 0) {
			t.Errorf("compare result invalid")
		}
	})
}

func TestSetVersion(t *testing.T) {
	t.Run("pyproject.toml", func(t *testing.T) {
		dir, _ := os.MkdirTemp("", "")
		file := filepath.Join(dir, "pyproject.toml")
		data := map[string]any{
			"project": map[string]any{
				"version": "0.0.0",
			},
		}
		fileBytes, _ := toml.Marshal(data)
		os.WriteFile(file, fileBytes, 0644)
		err := SetVersion(file, "1.0.0")
		if err != nil {
			t.Errorf("set version failed")
		}
		fileBytes, _ = os.ReadFile(file)
		toml.Unmarshal(fileBytes, &data)
		version := data["project"].(map[string]any)["version"]
		if version != "1.0.0" {
			t.Errorf("version mismatch %s", version)
		}
	})

	t.Run("package.json", func(t *testing.T) {
		dir, _ := os.MkdirTemp("", "")
		file := filepath.Join(dir, "package.json")
		data := map[string]any{
			"version": "0.0.0",
		}
		fileBytes, _ := json.Marshal(data)
		os.WriteFile(file, fileBytes, 0644)
		err := SetVersion(file, "1.0.0")
		if err != nil {
			t.Errorf("set version failed")
		}
		fileBytes, _ = os.ReadFile(file)
		json.Unmarshal(fileBytes, &data)
		version := data["version"]
		if version != "1.0.0" {
			t.Errorf("version mismatch %s", version)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		dir, _ := os.MkdirTemp("", "")
		file := filepath.Join(dir, "unknown")
		fileBytes := []byte("")
		os.WriteFile(file, fileBytes, 0644)
		err := SetVersion(file, "1.0.0")
		if err == nil {
			t.Errorf("set version did not fail")
		}
	})
}
