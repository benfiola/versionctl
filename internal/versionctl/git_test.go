package versionctl

import (
	"testing"

	"github.com/go-git/go-git/v5"
)

func createVersionctlRepo() Git {
	repo, err := git.PlainOpen("../..")
	if err != nil {
		panic(err)
	}
	g := Git{
		Repo: repo,
	}
	return g
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("versionctl repo", func(t *testing.T) {
		vGit := createVersionctlRepo()
		branch, err := vGit.GetCurrentBranch()
		if err != nil {
			panic(err)
		}
		if !(branch != "") {
			t.Errorf("branch not found")
		}
	})
}

func TestIterCommits(t *testing.T) {
	t.Run("versionctl repo", func(t *testing.T) {
		vGit := createVersionctlRepo()
		commits := []Commit{}
		err := vGit.IterCommits("", func(c Commit) error {
			commits = append(commits, c)
			return nil
		})
		if err != nil {
			panic(err)
		}
		if !(len(commits) != 0) {
			t.Errorf("commits not found")
		}
	})
}

func TestListTags(t *testing.T) {
	t.Run("versionctl repo", func(t *testing.T) {
		vGit := createVersionctlRepo()
		tags, err := vGit.ListTags()
		if err != nil {
			panic(err)
		}
		if !(len(tags) != 0) {
			t.Errorf("tags not found")
		}
	})
}
