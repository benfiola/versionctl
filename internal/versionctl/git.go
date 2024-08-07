package versionctl

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Commit struct {
	Hash    string
	Message string
	Tags    []string
}

type Git struct {
	Repo *git.Repository
}

func NewGit(path string) (Git, error) {
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Git{}, err
		}
		path = wd
	}
	repo, err := git.PlainOpen(path)
	if err != nil {
		return Git{}, err
	}
	return Git{Repo: repo}, nil
}

func (g Git) GetCurrentBranch() (string, error) {
	head, err := g.Repo.Head()
	if err != nil {
		return "", err
	}
	branch := head.Name().Short()
	return branch, err
}

type StopIter struct{}

func (s *StopIter) Error() string {
	return "stop iteration"
}

func (g Git) IterCommits(from string, cb func(Commit) error) error {
	tagRefs, err := g.Repo.Tags()
	if err != nil {
		return err
	}

	commitTags := map[string][]string{}
	err = tagRefs.ForEach(func(t *plumbing.Reference) error {
		commit, err := g.Repo.ResolveRevision(plumbing.Revision(t.Name()))
		if err != nil {
			return err
		}
		hash := commit.String()
		tagName := t.Name().Short()
		commitTags[hash] = append(commitTags[hash], tagName)
		return nil
	})
	if err != nil {
		return err
	}

	if from == "" {
		currentHead, err := g.Repo.Head()
		if err != nil {
			return err
		}
		from = currentHead.Hash().String()
	}

	fromHash := plumbing.NewHash(from)
	commitIterator, err := g.Repo.Log(&git.LogOptions{From: fromHash})
	if err != nil {
		return err
	}

	err = commitIterator.ForEach(func(c *object.Commit) error {
		hash := c.Hash.String()
		message := c.Message
		tags := commitTags[hash]

		commit := Commit{
			Hash:    hash,
			Message: message,
			Tags:    tags,
		}
		err = cb(commit)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (g Git) ListTags() ([]string, error) {
	tagRefs, err := g.Repo.Tags()
	if err != nil {
		return []string{}, err
	}

	tags := []string{}
	err = tagRefs.ForEach(func(t *plumbing.Reference) error {
		tagName := t.Name().Short()
		tags = append(tags, tagName)
		return nil
	})
	if err != nil {
		return []string{}, err
	}
	return tags, nil
}
