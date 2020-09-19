package gomk

import (
	"os"
	"path/filepath"
)

type Project struct {
	path string
}

func NewProject(dir string) Project {
	if dir == "" {
		dir, _ = os.Getwd() // TODO error
	} else if filepath.IsAbs(dir) {
		dir = filepath.Clean(dir)
	} else {
		wd, _ := os.Getwd() // TODO error
		dir = filepath.Join(wd, dir)
	}
	return Project{path: dir}
}

func (d Project) AbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(d.path, path)
}

func (d Project) RelPath(path string) string {
	if filepath.IsAbs(path) {
		res, err := filepath.Rel(d.path, path)
		if err != nil {
			return ""
		}
		return res
	}
	return path
}
