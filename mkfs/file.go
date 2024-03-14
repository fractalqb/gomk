package mkfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type File string

var _ Artefact = File("")

func (f File) Path() string { return string(f) }

func (f File) Name(in *gomkore.Project) string {
	n, _ := in.RelPath(f.Path())
	return n
}

func (f File) StateAt(in *gomkore.Project) (time.Time, error) {
	ap, err := in.AbsPath(f.Path())
	if err != nil {
		return time.Time{}, err
	}
	st, err := os.Stat(ap)
	switch {
	case err != nil:
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	case st.IsDir():
		return time.Time{}, fmt.Errorf("artefact %s is a directory", f.Path())
	}
	return st.ModTime(), nil
}

func (f File) Exists(in *gomkore.Project) (bool, error) {
	ap, err := in.AbsPath(f.Path())
	if err != nil {
		return false, err
	}
	st, err := os.Stat(ap)
	switch {
	case err == nil:
		if st.IsDir() {
			return true, fmt.Errorf("%s is a directory", f.Path())
		}
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	}
	return false, err
}

func (f File) Remove(in *gomkore.Project) error {
	ap, err := in.AbsPath(f.Path())
	if err != nil {
		return err
	}
	return os.Remove(ap)
}

func (f File) Moved(strip, dest Directory) (File, error) {
	var path string
	if strip == nil {
		var err error
		path, err = fsMove(f.Path(), "", dest.Path())
		if err != nil {
			return File(""), err
		}
	} else {
		var err error
		path, err = fsMove(f.Path(), strip.Path(), dest.Path())
		if err != nil {
			return File(""), err
		}
	}
	return File(filepath.ToSlash(path)), nil
}

func (f File) Ext() string { return filepath.Ext(f.Path()) }

func (f File) WithExt(ext string) File {
	path := f.Path()
	if ext == "" {
		ext = filepath.Ext(path)
		if ext == "" {
			return f
		}
		return File(path[:len(path)-len(ext)])
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	fExt := filepath.Ext(path)
	if fExt == "" {
		return File(path + ext)
	}
	return File(path[:len(path)-len(fExt)] + ext)
}
