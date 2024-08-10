package versionctl

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// A GitClient represents a git client.
type Git struct {
	repo *git.Repository
}

// Gets the current branch for the local working copy.
func (g Git) GetCurrentBranch() (string, error) {
	h, err := g.repo.Head()
	if err != nil {
		return "", err
	}
	return h.Name().Short(), nil
}

// A GitCommit represents data fields attached to a git commit
// within the local working copy
type GitCommit struct {
	Hash    string
	Message string
	Tags    []string
}

// Stops iteration when returned within an iteration callback
type StopIter struct {
}

// [StopIter] error interface implementation
func (s *StopIter) Error() string {
	return "stop iteration"
}

// Iterates through all commits from the provided head in reverse order.
// The callback is called for each [GitCommit] found.
// Return &StopIter{} to stop iteration.
// If head is a zero value, will use the current head of the local working copy
func (g Git) IterCommits(head string, cb func(c GitCommit) error) error {
	// use current head if not defined
	if head == "" {
		hd, err := g.repo.Head()
		if err != nil {
			return err
		}
		head = hd.Name().String()
	}
	// resolve hash of head
	hh, err := g.repo.ResolveRevision(plumbing.Revision(head))
	if err != nil {
		return err
	}
	// create hash -> tag[] map
	htm := map[string]([]string){}
	ts, err := g.repo.Tags()
	if err != nil {
		return err
	}
	ts.ForEach(func(t *plumbing.Reference) error {
		th := t.Hash().String()
		htm[th] = append(htm[th], t.Name().Short())
		return nil
	})
	// obtain commit iterator
	ci, err := g.repo.Log(&git.LogOptions{From: *hh})
	if err != nil {
		return err
	}
	// iterate over commits, create GitCommit objects, invoke callback
	err = ci.ForEach(func(oc *object.Commit) error {
		ch := oc.Hash.String()
		c := GitCommit{
			Hash:    ch,
			Message: oc.Message,
			Tags:    htm[ch],
		}
		err := cb(c)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		_, stopIter := err.(*StopIter)
		if !stopIter {
			return err
		}
	}
	return nil
}

// Lists all tags for the local working copy
func (g Git) ListTags() ([]string, error) {
	// obtain tag iterator
	i, err := g.repo.Tags()
	if err != nil {
		return []string{}, err
	}
	// iterate over and collect all tag names
	t := []string{}
	err = i.ForEach(func(r *plumbing.Reference) error {
		t = append(t, r.Name().Short())
		return nil
	})
	if err != nil {
		return []string{}, err
	}
	return t, nil
}

// Constructs a [Git].
// Accepts a path representing the local working copy.
// If path is a zero value, uses the process' current working directory.
func NewGit(path string) (Git, error) {
	// use current working directory if path is zero value
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Git{}, err
		}
		path = wd
	}
	r, err := git.PlainOpen(path)
	if err != nil {
		return Git{}, err
	}
	return Git{repo: r}, nil
}
