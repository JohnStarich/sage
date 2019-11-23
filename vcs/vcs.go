package vcs

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/johnstarich/sage/pipe"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// Repository is a Git repository with thread-safe file operations
type Repository interface {
	// CommitFiles commits with 'message' for files specified by 'paths'. 'prepFiles' is given exclusive access to files during execution
	CommitFiles(prepFiles func() error, message string, paths ...string) error
	// File returns a version-controlled file, capable of writing and committing in one operation
	File(path string) File
}

// Open ensures a Git repo exists at 'path' and returns its Repository
func Open(path string) (Repository, error) {
	path = filepath.Clean(path)
	if err := os.MkdirAll(path, 0750); err != nil {
		return nil, err
	}
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

func initVCS(path string) (*git.Repository, error) {
	var err error
	var repo *git.Repository
	var tree *git.Worktree
	var status git.Status
	return repo, pipe.OpFuncs{
		func() error {
			repo, err = git.PlainInit(path, false)
			return err
		},
		func() error {
			tree, err = repo.Worktree()
			return err
		},
		func() error {
			status, err = tree.Status()
			return err
		},
		func() error {
			var ops pipe.OpFuncs
			added := false
			for file, stat := range status {
				// add any untracked files, excluding hidden and tmp files
				if stat.Worktree == git.Untracked && !strings.HasPrefix(file, ".") && !strings.HasSuffix(file, ".tmp") {
					fileCopy := file
					ops = append(ops, func() error {
						_, err := tree.Add(fileCopy)
						return err
					})
					added = true
				}
			}
			if added {
				ops = append(ops, func() error {
					_, err := tree.Commit("Initial commit", &git.CommitOptions{Author: sageAuthor()})
					return err
				})
			}
			return ops.Do()
		},
	}.Do()
}

func sageAuthor() *object.Signature {
	return &object.Signature{
		Name: "Sage",
		When: time.Now(),
	}
}

// CommitFiles resets the repo index, then adds & commits the files at 'paths' with the 'message'
// Gives exclusive lock to 'prepFiles' execution.
func (s *syncRepo) CommitFiles(prepFiles func() error, message string, paths ...string) error {
	if len(paths) == 0 {
		return errors.New("No files to commit")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	var tree *git.Worktree
	var repoStatus git.Status
	var rootPath string
	return pipe.OpFuncs{
		prepFiles,
		func() error {
			tree, err = s.repo.Worktree()
			return err
		},
		func() error {
			_, headErr := s.repo.Head()
			if headErr != nil && headErr != plumbing.ErrReferenceNotFound {
				return headErr
			}
			if headErr != plumbing.ErrReferenceNotFound {
				// if possible (HEAD exists), unstage all files
				return tree.Reset(&git.ResetOptions{})
			}
			return nil
		},
		func() error {
			rootPath, err = filepath.Abs(tree.Filesystem.Root())
			return err
		},
		func() error {
			var ops pipe.OpFuncs
			for i := range paths {
				path := &paths[i]
				ops = append(ops,
					func() error {
						*path, err = filepath.Abs(*path)
						return err
					},
					func() error {
						*path, err = filepath.Rel(rootPath, *path)
						return err
					},
					func() error {
						_, err := tree.Add(*path)
						return errors.Wrapf(err, "Failed to add %s to the git index", *path)
					},
				)
			}
			return ops.Do()
		},
		func() error {
			repoStatus, err = tree.Status()
			return err
		},
		func() error {
			shouldCommit := false
			for _, path := range paths {
				status, ok := repoStatus[path]
				if ok && status.Staging != git.Unmodified {
					shouldCommit = true
					break
				}
			}
			if !shouldCommit {
				return nil
			}

			_, err = tree.Commit(message, &git.CommitOptions{
				Author: sageAuthor(),
			})
			return err
		},
	}.Do()

}
