package vcs

import (
	"io/ioutil"
	"os"
)

type File interface {
	Write(b []byte) error
	Read() ([]byte, error)
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

func (f *file) Read() ([]byte, error) {
	buf, err := ioutil.ReadFile(f.path)
	if os.IsNotExist(err) {
		err = nil
	}
	return buf, err
}

func diskWriter(path string, b []byte) func() error {
	return func() error {
		return ioutil.WriteFile(path, b, 0750)
	}
}
