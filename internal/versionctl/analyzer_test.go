package versionctl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type AnalyzerTestData struct {
	Analyzer *Analyzer
	Repo     *TestRepo
}

func createAnalyzerTestData(t testing.TB) *AnalyzerTestData {
	t.Helper()
	require := require.New(t)
	wd, err := os.Getwd()
	require.Nil(err)
	d, r := createGitRepo(t)
	os.Chdir(d)
	t.Cleanup(func() {
		os.Chdir(wd)
	})
	r.createGitCommit("initial")

	g, err := NewGit(&GitOpts{
		Path: d,
	})
	require.Nil(err)
	p, err := NewParser("default", &ParserOpts{
		BreakingChangeTags: []string{"breaking:"},
		Tags: map[string]string{
			"patch:": "patch",
			"minor:": "minor",
			"major:": "major",
		},
	})
	require.Nil(err)
	a, err := NewAnalyzer(&AnalyzerOpts{
		Git:    g,
		Parser: p,
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
	})
	require.Nil(err)
	return &AnalyzerTestData{
		Analyzer: a,
		Repo:     r,
	}
}

func TestAnalyzerGetCurrentVersion(t *testing.T) {
	t.Run("defaults to 0.0.0", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)

		v, err := td.Analyzer.GetCurrentVersion()

		require.Nil(err)
		require.Equal(Version{}, v)
	})

	t.Run("gets latest tag", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		td.Repo.createGitTag("v1.0.0")
		td.Repo.createGitTag("v0.0.1")
		td.Repo.createGitTag("1.0.0-rc.1")

		v, err := td.Analyzer.GetCurrentVersion()

		require.Nil(err)
		require.Equal(Version{Major: 1}, v)
	})
}

func TestAnalyzerGetNextVersion(t *testing.T) {
	t.Run("prerelease branch, repo version diff < change", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		// branch = prerelease
		td.Repo.checkoutGitBranch("dev")
		// repo version diff = patch (0.0.2 <-> 0.0.1)
		td.Repo.createGitTag("v0.0.2")
		td.Repo.createGitCommit("next")
		td.Repo.createGitTag("v0.0.1")
		// change = major
		td.Repo.createGitCommit("major: commit")

		// expected: version bump due to major change + prerelease data
		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Major: 1, Prerelease: Prerelease{Token: "rc", Count: 1}}, v)
	})

	t.Run("prerelease branch, repo version diff > change", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		// branch = prerelease
		td.Repo.checkoutGitBranch("dev")
		// repo version diff = minor (0.2.0 <-> 0.1.0)
		td.Repo.createGitTag("v0.2.0")
		td.Repo.createGitCommit("next")
		td.Repo.createGitTag("v0.1.0")
		// change = patch
		td.Repo.createGitCommit("patch: commit")

		// expected: no bump (reuse repo version) + prerelease data
		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2, Prerelease: Prerelease{Token: "rc", Count: 1}}, v)
	})

	t.Run("prerelease branch, reset prerelease count", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		// branch = prerelease
		td.Repo.checkoutGitBranch("dev")
		// repo version does not match prerelease token
		// repo version diff = minor (0.2.0-other.1 <-> 0.1.0)
		td.Repo.createGitTag("v0.2.0-other.1")
		td.Repo.createGitCommit("next")
		td.Repo.createGitTag("v0.1.0")
		// change = patch
		td.Repo.createGitCommit("patch: commit")

		// expected: no bump (reuse repo version) + reset prerelease count
		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2, Prerelease: Prerelease{Token: "rc", Count: 1}}, v)
	})

	t.Run("prerelease branch, increment prerelease count", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		// branch = prerelease
		td.Repo.checkoutGitBranch("dev")
		// repo version matching prerelease token
		// repo version diff = minor (0.2.0-rc.1 <-> 0.1.0)
		td.Repo.createGitTag("v0.2.0-rc.1")
		td.Repo.createGitCommit("next")
		td.Repo.createGitTag("v0.1.0")
		// change = patch
		td.Repo.createGitCommit("patch: commit")

		// expected: no bump (reuse repo version) + increment prerelease count
		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2, Prerelease: Prerelease{Token: "rc", Count: 2}}, v)
	})

	t.Run("release, repo version release", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		// branch = release
		td.Repo.checkoutGitBranch("main")
		// repo version is release
		td.Repo.createGitTag("v0.1.0")
		// change = patch
		td.Repo.createGitCommit("patch: commit")

		// expected: version bump from repo version
		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 1, Patch: 1}, v)
	})

	t.Run("release, repo version prerelease, repo version diff < change", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		// branch = release
		td.Repo.checkoutGitBranch("main")
		// repo version prerelease, repo version diff = minor
		td.Repo.createGitTag("v0.2.0-rc.1")
		td.Repo.createGitCommit("next")
		td.Repo.createGitTag("v0.1.0")
		// change = major
		td.Repo.createGitCommit("major: commit")

		// expected: repo version bump from change
		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Major: 1}, v)
	})

	t.Run("release, repo version prerelease, diff > change", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		// branch = release
		td.Repo.checkoutGitBranch("main")
		// repo version prerelease, repo version diff = minor
		td.Repo.createGitTag("v0.2.0-rc.1")
		td.Repo.createGitCommit("next")
		td.Repo.createGitTag("v0.1.0")
		// change = patch
		td.Repo.createGitCommit("patch: commit")

		// expected: convert repo version to release and use
		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Minor: 2}, v)
	})

	t.Run("metadata + capture groups", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		td.Repo.checkoutGitBranch("other/branch")
		td.Repo.createGitCommit("patch: initial")

		v, err := td.Analyzer.GetNextVersion()

		require.Nil(err)
		require.Equal(Version{Patch: 1, Prerelease: Prerelease{Token: "other-branch", Count: 1}, Metadata: "other-branch"}, v)
	})

	t.Run("fail if no change", func(t *testing.T) {
		require := require.New(t)
		td := createAnalyzerTestData(t)
		td.Repo.checkoutGitBranch("main")

		_, err := td.Analyzer.GetNextVersion()

		require.ErrorContains(err, "version unchanged")
	})
}
