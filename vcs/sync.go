package vcs

import (
	"io/ioutil"
)

type File interface {
	Write(b []byte) error
}

type file struct {
	path string
	repo Repository
}

func (repo *syncRepo) File(path string) File {
	return &file{
		path: path,
		repo: repo,
	}
}

func (f *file) Write(b []byte) error {
	return f.repo.CommitFiles(diskWriter(f.path, b), "Update "+f.path, f.path)
}

func diskWriter(path string, b []byte) func() error {
	return func() error {
		return ioutil.WriteFile(path, b, 0750)
	}
}
