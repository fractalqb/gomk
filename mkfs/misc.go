package mkfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type Artefact interface {
	gomkore.RemovableArtefact
	Path() string
}

func Stat(a Artefact, in *gomkore.Project) (fs.FileInfo, error) {
	p, err := in.AbsPath(a.Path())
	if err != nil {
		return nil, err
	}
	return os.Stat(p)
}

func Exists(a Artefact, in *gomkore.Project) (bool, error) {
	_, err := Stat(a, in)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func Moved(a Artefact, strip, dest Directory) (Artefact, error) {
	switch a := a.(type) {
	case File:
		return a.Moved(strip, dest)
	case DirList:
		return a.Moved(strip, dest)
	case DirTree:
		return a.Moved(strip, dest)
	}
	return a, fmt.Errorf("cannot move FsArtefact %T", a)
}

type Directory interface {
	Artefact
	List(in *gomkore.Project) ([]string, error)

	ls(string, func(string, fs.DirEntry) error) error
}

func movedPath(path, strip, dest string) (string, error) {
	if strip != "" {
		var err error
		if path, err = filepath.Rel(strip, path); err != nil {
			return "", err
		}
	}
	return filepath.Join(dest, path), nil
}

func rmDirIfEmpty(path string) error {
	if ok, err := isDirEmpty(path); err != nil {
		return err
	} else if !ok {
		return nil
	}
	return os.Remove(path)
}

func isDirEmpty(path string) (bool, error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer dir.Close()
	if _, err = dir.ReadDir(1); errors.Is(err, io.EOF) {
		return true, nil
	}
	return false, err
}
