package mkfs

import (
	"crypto/md5"
	"encoding/hex"
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
	prjDir, err := in.AbsPath(d.Path())
	if err != nil {
		return nil, err
	}
	err = d.ls(prjDir, func(_ string, e fs.DirEntry) error {
		ls = append(ls, filepath.Join(d.Dir, e.Name()))
		return nil
	})
	return
}

func (d DirList) Contains(in *gomkore.Project, a Artefact) (bool, error) {
	aDir := filepath.Dir(a.Path())
	dPath := filepath.Clean(d.Path())
	if aDir != dPath || d.Filter == nil {
		return false, nil
	}
	aPrj, err := in.AbsPath(a.Path())
	if err != nil {
		return false, err
	}
	stat, err := os.Stat(aPrj)
	if err != nil {
		return false, err
	}
	ok, err := d.Filter.Ok(a.Path(), infoEntry{FileInfo: stat})
	if errors.Is(err, fs.SkipDir) {
		err = nil
	}
	return ok, err
}

func (d DirList) Goals(in *gomkore.Project) (gs []*gomkore.Goal, err error) {
	prjDir, err := in.AbsPath(d.Path())
	if err != nil {
		return nil, err
	}
	err = d.ls(prjDir, func(_ string, e fs.DirEntry) error {
		p := filepath.Join(d.Dir, e.Name())
		if e.IsDir() {
			dir := DirList{Dir: p, Filter: d.Filter}
			g, err := in.Goal(dir)
			if err != nil {
				return err
			}
			gs = append(gs, g)
		} else {
			g, err := in.Goal(File(p))
			if err != nil {
				return err
			}
			gs = append(gs, g)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gs, nil
}

type dirListKey string

func (d DirList) Key() any {
	h := md5.New() // No crypto relevance! (?)
	fmt.Fprintln(h, d.Dir)
	if d.Filter != nil {
		d.Filter.Hash(h)
	}
	return dirListKey(hex.EncodeToString(h.Sum(nil)))
}

func (d DirList) Name(prj *gomkore.Project) string {
	n, _ := prj.RelPath(d.Dir)
	return n
}

func (d DirList) StateAt(in *gomkore.Project) (t time.Time, err error) {
	prjDir, err := in.AbsPath(d.Path())
	if err != nil {
		return time.Time{}, err
	}
	err = d.ls(prjDir, func(_ string, e fs.DirEntry) error {
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
	prjDir, err := in.AbsPath(d.Path())
	if err != nil {
		return err
	}
	err = d.ls(prjDir, func(_ string, e fs.DirEntry) error {
		p := filepath.Join(prjDir, e.Name())
		return os.Remove(p)
	})
	if err != nil {
		return err
	}
	return rmDirIfEmpty(prjDir)
}

func (d DirList) Moved(strip, dest Directory) (DirList, error) {
	var path string
	if strip == nil {
		var err error
		path, err = movedPath(d.Path(), "", dest.Path())
		if err != nil {
			return DirList{}, err
		}
	} else {
		var err error
		path, err = movedPath(d.Path(), strip.Path(), dest.Path())
		if err != nil {
			return DirList{}, err
		}
	}
	return DirList{
		Dir:    filepath.ToSlash(path),
		Filter: d.Filter,
	}, nil
}

func (d DirList) ls(prjDir string, do func(p string, e fs.DirEntry) error) error {
	rdir, err := os.ReadDir(prjDir)
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
		if err := do(entry.Name(), entry); err != nil {
			return err
		}
	}
	return nil
}
