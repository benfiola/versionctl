package version

import "testing"

func TestNew(t *testing.T) {
	t.Run("release", func(t *testing.T) {
		version, err := New("1.2.3")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease == nil && version.Metadata == nil) {
			t.Errorf("version invalid: %#v", version)
		}
	})

	t.Run("prerelease", func(t *testing.T) {
		version, err := New("1.2.3-rc.1")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease.Token == "rc" && version.Prerelease.Count == 1 && version.Metadata == nil) {
			t.Errorf("version invalid: %#v", version)
		}
	})

	t.Run("metadata", func(t *testing.T) {
		version, err := New("1.2.3+metadata")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease == nil && *version.Metadata == "metadata") {
			t.Errorf("version invalid: %#v", version)
		}
	})

	t.Run("prerelease and metadata", func(t *testing.T) {
		version, err := New("1.2.3-rc.1+metadata")
		if !(err == nil && version.Major == 1 && version.Minor == 2 && version.Patch == 3 && version.Prerelease.Token == "rc" && version.Prerelease.Count == 1 && *version.Metadata == "metadata") {
			t.Errorf("version invalid: %#v", version)
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
