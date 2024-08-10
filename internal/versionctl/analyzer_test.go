package versionctl

import (
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"
)

var testConfig = Config{
	BreakingChangeTags: []string{"breaking:"},
	Rules: []Rule{
		{
			Branch: "main",
		},
		{
			Branch:          "dev",
			PrereleaseToken: "rc",
		},
		{
			Branch:          "(?P<branch>.*)",
			PrereleaseToken: "{branch}",
			Metadata:        "{branch}",
		},
	},
	Tags: map[string]string{
		"patch:": "patch",
		"minor:": "minor",
		"major:": "major",
	},
}

// Creates a base repository for use with analyzer unit tests
func createAnalyzerRepo(t testing.TB) *git.Repository {
	t.Helper()
	require := require.New(t)
	wd, err := os.Getwd()
	require.Nil(err)
	d, r := createGitRepo(t)
	os.Chdir(d)
	t.Cleanup(func() {
		os.Chdir(wd)
	})
	createGitCommit(t, r, "initial")
	return r
}

func TestAnalyzerGetCurrentVersion(t *testing.T) {
	t.Run("defaults to 0.0.0", func(t *testing.T) {
		require := require.New(t)
		createAnalyzerRepo(t)
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		v, err := a.GetCurrentVersion()

		require.Nil(err)
		require.Equal(Version{}, v)
	})

	t.Run("gets latest tag", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		createGitTag(t, r, "v1.0.0")
		createGitTag(t, r, "v0.0.1")
		createGitTag(t, r, "1.0.0-rc.1")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		v, err := a.GetCurrentVersion()

		require.Nil(err)
		require.Equal(Version{Major: 1}, v)
	})
}

func TestAnalyzerGetNextVersion(t *testing.T) {
	t.Run("prerelease branch, repo version diff < change", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		// branch = prerelease
		checkoutGitBranch(t, r, "dev")
		// repo version diff = patch (0.0.2 <-> 0.0.1)
		createGitTag(t, r, "v0.0.2")
		createGitCommit(t, r, "next")
		createGitTag(t, r, "v0.0.1")
		// change = major
		createGitCommit(t, r, "major: commit")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		// expected: version bump due to major change + prerelease data
		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Major: 1, Prerelease: Prerelease{Token: "rc", Count: 1}}, v)
	})

	t.Run("prerelease branch, repo version diff > change", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		// branch = prerelease
		checkoutGitBranch(t, r, "dev")
		// repo version diff = minor (0.2.0 <-> 0.1.0)
		createGitTag(t, r, "v0.2.0")
		createGitCommit(t, r, "next")
		createGitTag(t, r, "v0.1.0")
		// change = patch
		createGitCommit(t, r, "patch: commit")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		// expected: no bump (reuse repo version) + prerelease data
		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2, Prerelease: Prerelease{Token: "rc", Count: 1}}, v)
	})

	t.Run("prerelease branch, reset prerelease count", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		// branch = prerelease
		checkoutGitBranch(t, r, "dev")
		// repo version does not match prerelease token
		// repo version diff = minor (0.2.0-other.1 <-> 0.1.0)
		createGitTag(t, r, "v0.2.0-other.1")
		createGitCommit(t, r, "next")
		createGitTag(t, r, "v0.1.0")
		// change = patch
		createGitCommit(t, r, "patch: commit")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		// expected: no bump (reuse repo version) + reset prerelease count
		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2, Prerelease: Prerelease{Token: "rc", Count: 1}}, v)
	})

	t.Run("prerelease branch, increment prerelease count", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		// branch = prerelease
		checkoutGitBranch(t, r, "dev")
		// repo version matching prerelease token
		// repo version diff = minor (0.2.0-rc.1 <-> 0.1.0)
		createGitTag(t, r, "v0.2.0-rc.1")
		createGitCommit(t, r, "next")
		createGitTag(t, r, "v0.1.0")
		// change = patch
		createGitCommit(t, r, "patch: commit")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		// expected: no bump (reuse repo version) + increment prerelease count
		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2, Prerelease: Prerelease{Token: "rc", Count: 2}}, v)
	})

	t.Run("release, repo version release", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		// branch = release
		checkoutGitBranch(t, r, "main")
		// repo version is release
		createGitTag(t, r, "v0.1.0")
		// change = patch
		createGitCommit(t, r, "patch: commit")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		// expected: version bump from repo version
		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 1, Patch: 1}, v)
	})

	t.Run("release, repo version prerelease, repo version diff < change", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		// branch = release
		checkoutGitBranch(t, r, "main")
		// repo version prerelease, repo version diff = minor
		createGitTag(t, r, "v0.2.0-rc.1")
		createGitCommit(t, r, "next")
		createGitTag(t, r, "v0.1.0")
		// change = major
		createGitCommit(t, r, "major: commit")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		// expected: repo version bump from change
		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Major: 1}, v)
	})

	t.Run("release, repo version prerelease, diff > change", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		// branch = release
		checkoutGitBranch(t, r, "main")
		// repo version prerelease, repo version diff = minor
		createGitTag(t, r, "v0.2.0-rc.1")
		createGitCommit(t, r, "next")
		createGitTag(t, r, "v0.1.0")
		// change = patch
		createGitCommit(t, r, "patch: commit")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		// expected: convert repo version to release and use
		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2}, v)
	})

	t.Run("metadata + capture groups", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		checkoutGitBranch(t, r, "other/branch")
		createGitCommit(t, r, "patch: initial")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		v, err := a.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Patch: 1, Prerelease: Prerelease{Token: "other-branch", Count: 1}, Metadata: "other-branch"}, v)
	})

	t.Run("fail if no change", func(t *testing.T) {
		require := require.New(t)
		r := createAnalyzerRepo(t)
		checkoutGitBranch(t, r, "main")
		a, err := NewAnalyzer(testConfig)
		require.Nil(err)

		_, err = a.GetNextVersion()

		require.ErrorContains(err, "version unchanged")
	})
}
