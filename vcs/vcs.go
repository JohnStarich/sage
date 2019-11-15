package vcs

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Repository interface {
	// CommitFiles commits with 'message' for files specified by 'paths'. 'prepFiles' is given exclusive access to files during execution
	CommitFiles(prepFiles func() error, message string, paths ...string) error
}

func Open(path string) (Repository, error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit: false,
	})
	if err == git.ErrRepositoryNotExists {
		repo, err = initVCS(path)
	}
	return &syncRepo{repo: repo}, err
}

type syncRepo struct {
	repo *git.Repository
	mu   sync.Mutex
}

func sageAuthor() *object.Signature {
	return &object.Signature{
		Name: "Sage",
		When: time.Now(),
	}
}

func initVCS(path string) (*git.Repository, error) {
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, err
	}
	tree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	status, err := tree.Status()
	if err != nil {
		return nil, err
	}

	added := false
	for file, stat := range status {
		// add any untracked bucket files
		if stat.Worktree == git.Untracked && strings.HasSuffix(file, ".json") {
			_, err := tree.Add(file)
			if err != nil {
				return nil, err
			}
			added = true
		}
	}
	if added {
		tree.Commit("Initial commit", &git.CommitOptions{
			Author: sageAuthor(),
		})
	}
	return repo, nil
}

// CommitFiles resets the repo index, then adds & commits the files at 'paths' with the 'message'
// Gives exclusive lock to 'prepFiles' execution.
func (s *syncRepo) CommitFiles(prepFiles func() error, message string, paths ...string) error {
	if len(paths) == 0 {
		return errors.New("No files to commit")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := prepFiles(); err != nil {
		return err
	}

	tree, err := s.repo.Worktree()
	if err != nil {
		return err
	}
	_, headErr := s.repo.Head()
	if headErr != nil && headErr != plumbing.ErrReferenceNotFound {
		return headErr
	}
	if headErr != plumbing.ErrReferenceNotFound {
		if err := tree.Reset(&git.ResetOptions{}); err != nil { // unstage everything
			return err
		}
	}
	rootPath := tree.Filesystem.Root()
	for _, path := range paths {
		path, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		if _, err = tree.Add(path); err != nil {
			return errors.Wrapf(err, "Failed to add %s to the git index", path)
		}
	}
	_, err = tree.Commit(message, &git.CommitOptions{
		Author: sageAuthor(),
	})
	return err
}
