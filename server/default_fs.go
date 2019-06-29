package server

import (
	"net/http"
	"os"
	"strings"
)

type defaultRouteFS struct {
	defaultFile    string
	fs             http.FileSystem
	staticPrefixes []string
}

// newDefaultRouteFS wraps an http.FileSystem and if a file isn't found, routes it to the defaultFile instead of failing
func newDefaultRouteFS(defaultFile string, fs http.FileSystem, staticPrefixes ...string) http.FileSystem {
	return &defaultRouteFS{
		defaultFile:    defaultFile,
		fs:             fs,
		staticPrefixes: staticPrefixes,
	}
}

func (d *defaultRouteFS) Open(name string) (http.File, error) {
	f, err := d.fs.Open(name)
	if err == os.ErrNotExist && d.isNotPrefixed(name) {
		return d.fs.Open(d.defaultFile)
	}
	return f, err
}

func (d *defaultRouteFS) isNotPrefixed(name string) bool {
	for _, dir := range d.staticPrefixes {
		if strings.HasPrefix(name, dir) {
			return false
		}
	}
	return true
}
