package mkfs

import (
	"fmt"
	"hash"
	"io/fs"
	"path/filepath"
	"strings"
)

type Filter interface {
	Ok(path string, entry fs.DirEntry) (bool, error)
	Hash(f hash.Hash)
}

type FilterFunc func(string, fs.DirEntry) (bool, error)

func (ff FilterFunc) Ok(p string, e fs.DirEntry) (bool, error) {
	return ff(p, e)
}

func (ff FilterFunc) Hash(h hash.Hash) {
	fmt.Fprintf(h, "mkfs.FilterFunc %p", ff)
}

type IsDir bool

func (d IsDir) Ok(_ string, e fs.DirEntry) (bool, error) {
	return e.IsDir() == bool(d), nil
}

func (d IsDir) Hash(h hash.Hash) {
	fmt.Fprintf(h, "mkfs.IsDir %t", d)
}

type NameMatch string

func (p NameMatch) Ok(_ string, e fs.DirEntry) (bool, error) {
	return filepath.Match(string(p), e.Name())
}

func (p NameMatch) Hash(h hash.Hash) {
	fmt.Fprintf(h, "mkfs.NameMatch %s", p)
}

type exts map[string]bool

func Exts(list ...string) exts {
	res := make(exts, len(list))
	for _, ext := range list {
		if !strings.HasPrefix(ext, ".") {
			res["."+ext] = true
		} else {
			res[ext] = true
		}
	}
	return res
}

func (fx exts) Ok(_ string, e fs.DirEntry) (bool, error) {
	ext := filepath.Ext(e.Name())
	return fx[ext], nil
}

func (fx exts) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.Exts")
	for k := range fx {
		fmt.Fprintln(h, k)
	}
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

func (fm Mode) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.Mode")
	fmt.Fprintln(h, fm.All)
	fmt.Fprintln(h, fm.Any)
}

type MaxPathLen int

func (fp MaxPathLen) Ok(p string, _ fs.DirEntry) (bool, error) {
	parts := strings.Split(p, string(filepath.Separator))
	return len(parts) <= int(fp), nil
}

func (fp MaxPathLen) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.MaxPathLen")
	fmt.Fprintln(h, fp)
}

type Not struct{ Filter }

func (fn Not) Ok(p string, e fs.DirEntry) (bool, error) {
	ok, err := fn.Filter.Ok(p, e)
	return !ok, err
}

func (fn Not) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.Not")
	fn.Filter.Hash(h)
}

type skipPaths map[string]bool

func SkipPaths(paths ...string) skipPaths {
	res := make(skipPaths, len(paths))
	for _, n := range paths {
		res[n] = true
	}
	return res
}

func (sp skipPaths) Ok(p string, e fs.DirEntry) (bool, error) {
	if !e.IsDir() {
		return true, nil
	}
	if sp[p] {
		return false, fs.SkipDir
	}
	return true, nil
}

func (sp skipPaths) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.SkipPaths")
	for p := range sp {
		fmt.Fprintln(h, p)
	}
}

type skipNames map[string]bool

func SkipNames(names ...string) skipNames {
	res := make(skipNames, len(names))
	for _, n := range names {
		res[n] = true
	}
	return res
}

func (sn skipNames) Ok(_ string, e fs.DirEntry) (bool, error) {
	if !e.IsDir() {
		return true, nil
	}
	if sn[e.Name()] {
		return false, fs.SkipDir
	}
	return true, nil
}

func (sn skipNames) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.SkipNames")
	for p := range sn {
		fmt.Fprintln(h, p)
	}
}

type SkipMatch string

func (sm SkipMatch) Ok(_ string, e fs.DirEntry) (bool, error) {
	if ok, err := filepath.Match(string(sm), e.Name()); err != nil {
		return false, err
	} else if ok {
		if e.IsDir() {
			return false, fs.SkipDir
		}
		return false, nil
	}
	return true, nil
}

func (sm SkipMatch) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.SkipMatch", sm)
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

func (fs All) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.All")
	for _, f := range fs {
		f.Hash(h)
	}
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

func (fs Any) Hash(h hash.Hash) {
	fmt.Fprintln(h, "mkfs.Any")
	for _, f := range fs {
		f.Hash(h)
	}
}
