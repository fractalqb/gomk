package mkfs

import (
	"io/fs"
	"path/filepath"
	"strings"
)

type Filter interface {
	Ok(path string, entry fs.DirEntry) (bool, error)
}

type FilterFunc func(string, fs.DirEntry) (bool, error)

func (ff FilterFunc) Ok(p string, e fs.DirEntry) (bool, error) {
	return ff(p, e)
}

type IsDir bool

func (d IsDir) Ok(_ string, e fs.DirEntry) (bool, error) {
	return e.IsDir() == bool(d), nil
}

type NameMatch string

func (p NameMatch) Ok(_ string, e fs.DirEntry) (bool, error) {
	return filepath.Match(string(p), e.Name())
}

type Mode struct{ Any, All fs.FileMode }

func (fm Mode) Ok(_ string, e fs.DirEntry) (bool, error) {
	info, err := e.Info()
	if err != nil {
		return false, err
	}
	mode, ok := info.Mode(), true
	if fm.Any != 0 {
		ok = ok && mode&fm.Any != 0
	}
	if fm.All != 0 {
		ok = ok && mode&fm.All == fm.All
	}
	return ok, nil
}

type MaxPathLen int

func (fp MaxPathLen) Ok(p string, _ fs.DirEntry) (bool, error) {
	parts := strings.Split(p, string(filepath.Separator))
	return len(parts) <= int(fp), nil
}

func Not(f Filter) Filter {
	return FilterFunc(func(p string, e fs.DirEntry) (bool, error) {
		ok, err := f.Ok(p, e)
		return !ok, err
	})
}

type All []Filter

func (fs All) Ok(p string, e fs.DirEntry) (bool, error) {
	for _, f := range fs {
		if ok, err := f.Ok(p, e); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

type Any []Filter

func (fs Any) Ok(p string, e fs.DirEntry) (bool, error) {
	for _, f := range fs {
		if ok, err := f.Ok(p, e); err != nil {
			return ok, err
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}
