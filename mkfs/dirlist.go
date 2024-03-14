package mkfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type DirList struct {
	Dir    string
	Filter Filter
}

var _ Directory = DirList{}

func (d DirList) Path() string { return d.Dir }

func (d DirList) List(in *gomkore.Project) (ls []string, err error) {
	prjDir, err := in.AbsPath(d.Dir)
	if err != nil {
		return nil, err
	}
	err = d.ls(prjDir, func(e fs.DirEntry) error {
		ls = append(ls, filepath.Join(d.Dir, e.Name()))
		return nil
	})
	return
}

func (d DirList) Name(prj *gomkore.Project) string {
	n, _ := prj.RelPath(d.Dir)
	return n
}

func (d DirList) StateAt(in *gomkore.Project) (t time.Time, err error) {
	prjDir, err := in.AbsPath(d.Dir)
	if err != nil {
		return time.Time{}, err
	}
	err = d.ls(prjDir, func(e fs.DirEntry) error {
		p := filepath.Join(prjDir, e.Name())
		if st, err := os.Stat(p); err != nil {
			return err
		} else if mt := st.ModTime(); mt.After(t) {
			t = mt
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func (d DirList) Exists(in *gomkore.Project) (bool, error) {
	ap, err := in.AbsPath(d.Path())
	if err != nil {
		return false, err
	}
	st, err := os.Stat(ap)
	switch {
	case err == nil:
		if !st.IsDir() {
			return true, fmt.Errorf("%s is no directory", d.Path())
		}
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	}
	return false, err
}

func (d DirList) Remove(in *gomkore.Project) error {
	prjDir, err := in.AbsPath(d.Dir)
	if err != nil {
		return err
	}
	return d.ls(prjDir, func(de fs.DirEntry) error {
		f := filepath.Join(prjDir, de.Name())
		return os.Remove(f)
	})
}

func (d DirList) Moved(strip, dest Directory) (DirList, error) {
	var path string
	if strip == nil {
		var err error
		path, err = fsMove(d.Path(), "", dest.Path())
		if err != nil {
			return DirList{}, err
		}
	} else {
		var err error
		path, err = fsMove(d.Path(), strip.Path(), dest.Path())
		if err != nil {
			return DirList{}, err
		}
	}
	return DirList{
		Dir:    filepath.ToSlash(path),
		Filter: d.Filter,
	}, nil
}

func (d DirList) ls(dir string, do func(fs.DirEntry) error) error {
	rdir, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range rdir {
		if d.Filter != nil {
			if ok, err := d.Filter.Ok(entry.Name(), entry); err != nil {
				return err
			} else if !ok {
				continue
			}
		}
		if err := do(entry); err != nil {
			return err
		}
	}
	return nil
}
