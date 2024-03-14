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

type DirTree struct {
	Dir    string
	Filter Filter
}

var _ Directory = DirTree{}

func DirFiles(dir, match string, pathMax int) DirTree {
	res := DirTree{Dir: dir}
	if match == "" {
		res.Filter = IsDir(true)
	} else {
		res.Filter = All{IsDir(true), NameMatch(match)}
	}
	if pathMax > 0 {
		switch es := res.Filter.(type) {
		case nil:
			res.Filter = MaxPathLen(pathMax)
		case All:
			res.Filter = append(es, MaxPathLen(pathMax))
		default:
			res.Filter = All{es, MaxPathLen(pathMax)}
		}
	}
	return res
}

func (d DirTree) Path() string { return d.Dir }

func (d DirTree) List(in *gomkore.Project) (ls []string, err error) {
	root, err := in.AbsPath(d.Path())
	if err != nil {
		return nil, err
	}
	err = d.ls(root, func(p string, e fs.DirEntry) error {
		p, err := in.RelPath(p)
		if err != nil {
			return err
		}
		ls = append(ls, p)
		return nil
	})
	return
}

func (d DirTree) Name(in *gomkore.Project) string {
	n, _ := in.RelPath(d.Dir)
	return n
}

func (d DirTree) StateAt(in *gomkore.Project) (t time.Time, err error) {
	root, err := in.AbsPath(d.Dir)
	if err != nil {
		return time.Time{}, err
	}
	err = d.ls(root, func(_ string, e fs.DirEntry) error {
		if info, err := e.Info(); err != nil {
			return err
		} else if mt := info.ModTime(); mt.After(t) {
			t = mt
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func (d DirTree) Exists(in *gomkore.Project) (bool, error) {
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

func (d DirTree) Remove(in *gomkore.Project) error {
	prjDir, err := in.AbsPath(d.Dir)
	if err != nil {
		return err
	}
	return d.ls(prjDir, func(p string, _ fs.DirEntry) error {
		return os.Remove(p)
	})
}

func (d DirTree) Moved(strip, dest Directory) (DirTree, error) {
	var path string
	if strip == nil {
		var err error
		path, err = fsMove(d.Path(), "", dest.Path())
		if err != nil {
			return DirTree{}, err
		}
	} else {
		var err error
		path, err = fsMove(d.Path(), strip.Path(), dest.Path())
		if err != nil {
			return DirTree{}, err
		}
	}
	return DirTree{
		Dir:    filepath.ToSlash(path),
		Filter: d.Filter,
	}, nil
}

func (d DirTree) ls(root string, do func(string, fs.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path, err = filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if ok, err := d.ok(path, e); err != nil {
			return err
		} else if ok {
			if err := do(path, e); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d DirTree) ok(p string, e fs.DirEntry) (ok bool, err error) {
	if d.Filter != nil {
		return d.Filter.Ok(p, e)
	}
	return true, nil
}
