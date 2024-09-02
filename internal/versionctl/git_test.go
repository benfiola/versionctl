package versionctl

import (
	"os"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

type TestRepo struct {
	*git.Repository
	t testing.TB
}

// Helper method to create a git repo.
func createGitRepo(t testing.TB) (string, *TestRepo) {
	t.Helper()
	require := require.New(t)
	d := t.TempDir()
	r, err := git.PlainInit(d, false)
	require.Nil(err)
	return d, &TestRepo{
		Repository: r,
		t:          t,
	}
}

// Helper method to create a git commit with the provided message
func (r *TestRepo) createGitCommit(message string) string {
	r.t.Helper()
	require := require.New(r.t)
	wt, err := r.Worktree()
	require.Nil(err)
	h, err := wt.Commit(message, &git.CommitOptions{AllowEmptyCommits: true, Author: &object.Signature{Name: "author", Email: "email", When: time.Now()}})
	require.Nil(err)
	return h.String()
}

// Helper method to checkout a git branch (creates a branch if it does not exist)
func (r *TestRepo) checkoutGitBranch(name string) {
	r.t.Helper()
	require := require.New(r.t)
	_, err := r.ResolveRevision(plumbing.Revision(name))
	c := err != nil
	wt, err := r.Worktree()
	require.Nil(err)
	err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(name), Create: c})
	require.Nil(err)
}

// Helper method to create a git tag at the current head.
func (r *TestRepo) createGitTag(name string) {
	r.t.Helper()
	require := require.New(r.t)
	h, err := r.Head()

	r.CreateTag(name, h.Hash(), nil)
	require.Nil(err)
}

func TestNewGit(t *testing.T) {
	t.Run("fails when not git repository", func(t *testing.T) {
		require := require.New(t)

		_, err := NewGit(&GitOpts{
			Path: t.TempDir(),
		})
		require.ErrorContains(err, "repository does not exist")
	})

	t.Run("uses cwd by default", func(t *testing.T) {
		require := require.New(t)
		wd, err := os.Getwd()
		require.Nil(err)
		d, _ := createGitRepo(t)
		err = os.Chdir(d)
		require.Nil(err)
		t.Cleanup(func() {
			os.Chdir(wd)
		})

		_, err = NewGit(&GitOpts{})
		require.Nil(err)
	})

	t.Run("success when git repository", func(t *testing.T) {
		require := require.New(t)
		d, _ := createGitRepo(t)

		g, err := NewGit(&GitOpts{
			Path: d,
		})

		require.Nil(err)
		require.NotNil(g)
	})
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("gets current branch", func(t *testing.T) {
		require := require.New(t)
		d, r := createGitRepo(t)
		r.createGitCommit("initial")

		g, err := NewGit(&GitOpts{
			Path: d,
		})
		require.Nil(err)

		b, err := g.GetCurrentBranch()

		require.Nil(err)
		require.Equal(b, "master")
	})
}

func TestIterCommits(t *testing.T) {
	t.Run("captures hash", func(t *testing.T) {
		require := require.New(t)
		d, r := createGitRepo(t)
		h := r.createGitCommit("message")

		g, err := NewGit(&GitOpts{
			Path: d,
		})
		require.Nil(err)

		commits := []GitCommit{}
		g.IterCommits("", func(c GitCommit) error {
			commits = append(commits, c)
			return nil
		})

		require.Equal(1, len(commits))
		require.Equal(h, commits[0].Hash)
	})

	t.Run("captures message", func(t *testing.T) {
		require := require.New(t)
		d, r := createGitRepo(t)
		r.createGitCommit("message")

		g, err := NewGit(&GitOpts{
			Path: d,
		})
		require.Nil(err)

		commits := []GitCommit{}
		g.IterCommits("", func(c GitCommit) error {
			commits = append(commits, c)
			return nil
		})

		require.Equal(1, len(commits))
		require.Equal("message", commits[0].Message)
	})

	t.Run("captures tags", func(t *testing.T) {
		require := require.New(t)
		d, r := createGitRepo(t)
		r.createGitCommit("no tags")
		r.createGitCommit("tags")
		r.createGitTag("tag")

		g, err := NewGit(&GitOpts{
			Path: d,
		})
		require.Nil(err)

		commits := []GitCommit{}
		g.IterCommits("", func(c GitCommit) error {
			commits = append(commits, c)
			return nil
		})

		require.Equal(2, len(commits))
		require.Equal("tags", commits[0].Message)
		require.Equal(1, len(commits[0].Tags))
		require.Equal("tag", commits[0].Tags[0])
		require.Equal("no tags", commits[1].Message)
		require.Equal(0, len(commits[1].Tags))
	})

	t.Run("iterates in descending order", func(t *testing.T) {
		require := require.New(t)
		d, r := createGitRepo(t)
		r.createGitCommit("a")
		r.createGitCommit("b")

		g, err := NewGit(&GitOpts{
			Path: d,
		})
		require.Nil(err)

		commits := []GitCommit{}
		g.IterCommits("", func(c GitCommit) error {
			commits = append(commits, c)
			return nil
		})

		require.Equal(2, len(commits))
		require.Equal("b", commits[0].Message)
		require.Equal("a", commits[1].Message)
	})

	t.Run("stop iteration", func(t *testing.T) {
		require := require.New(t)
		d, r := createGitRepo(t)
		r.createGitCommit("a")
		r.createGitCommit("b")

		g, err := NewGit(&GitOpts{
			Path: d,
		})
		require.Nil(err)

		commits := []GitCommit{}
		g.IterCommits("", func(c GitCommit) error {
			commits = append(commits, c)
			return &StopIter{}
		})

		require.Equal(1, len(commits))
		require.Equal("b", commits[0].Message)
	})
}
func TestListTags(t *testing.T) {
	t.Run("list tags", func(t *testing.T) {
		require := require.New(t)
		d, r := createGitRepo(t)
		r.createGitCommit("initial")

		g, err := NewGit(&GitOpts{
			Path: d,
		})
		require.Nil(err)

		ts, err := g.ListTags()
		require.Nil(err)
		require.Equal(0, len(ts))

		r.createGitTag("test")

		ts, err = g.ListTags()
		require.Nil(err)
		require.Equal(1, len(ts))
		require.Equal("test", ts[0])
	})
}
