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

type Mirror struct {
	Orig   Directory
	Strip  string
	Dest   Directory
	ExtMap map[string]string
}

var _ Artefact = Mirror{}

func (m Mirror) Key() any {
	h := md5.New()
	fmt.Fprint(h, m.Orig.Key(), m.Strip, m.Dest.Key())
	return mirrorKey(hex.EncodeToString(h.Sum(nil)))
}

type mirrorKey string

func (m Mirror) Name(in *gomkore.Project) string { return m.Dest.Name(in) }

func (m Mirror) StateAt(in *gomkore.Project) (t time.Time, err error) {
	err = m.ls(in, func(rel string) error {
		abs, err := in.AbsPath(rel)
		if err != nil {
			return err
		}
		st, err := os.Stat(abs)
		if err != nil {
			return err
		}
		if tm := st.ModTime(); t.Before(tm) {
			t = tm
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return time.Time{}, nil
	}
	return
}

func (m Mirror) Exists(in *gomkore.Project) (ok bool, err error) {
	t, err := m.StateAt(in)
	if err != nil {
		return false, err
	}
	return !t.IsZero(), nil
}

func (m Mirror) Remove(in *gomkore.Project) error {
	return m.ls(in, func(rel string) error {
		abs, err := in.AbsPath(rel)
		if err != nil {
			return err
		}
		return os.Remove(abs)
	})
}

func (m Mirror) Path() string { return m.Dest.Path() }

func (m Mirror) List(in *gomkore.Project) (ls []string, err error) {
	err = m.ls(in, func(rel string) error {
		ls = append(ls, rel)
		return nil
	})
	return
}

func (m Mirror) ls(in *gomkore.Project, do func(rel string) error) error {
	orig, err := in.AbsPath(m.Orig.Path())
	if err != nil {
		return err
	}
	var strip string
	if m.Strip == "" {
		if strip, err = in.AbsPath(""); err != nil {
			return err
		}
	} else if strip, err = in.AbsPath(m.Strip); err != nil {
		return err
	}
	switch dest := m.Dest.(type) {
	case DirList:
		return m.Orig.ls(orig, func(_ string, e fs.DirEntry) error {
			e = m.mapExtE(e)
			p := filepath.Join(dest.Path(), e.Name())
			if dest.Filter != nil {
				if ok, err := dest.Filter.Ok(p, e); err != nil {
					return err
				} else if ok {
					return do(p)
				}
				return nil
			} else {
				return do(p)
			}
		})
	case DirTree:
		return m.Orig.ls(orig, func(p string, e fs.DirEntry) error {
			e = m.mapExtE(e)
			p = m.mapExtP(p)
			p = filepath.Join(orig, p)
			rel, err := filepath.Rel(strip, p)
			if err != nil {
				return err
			}
			p = filepath.Join(dest.Path(), rel)
			if dest.Filter != nil {
				if ok, err := dest.Filter.Ok(p, e); err != nil {
					return err
				} else if ok {
					return do(p)
				}
				return nil
			} else {
				return do(p)
			}
		})
	}
	return fmt.Errorf("illegal mirror dest type %T", m.Dest)
}

type extMapEntry struct {
	fs.DirEntry
	name string
}

func (e extMapEntry) Name() string { return e.name }

func (m Mirror) mapExtE(e fs.DirEntry) fs.DirEntry {
	if m.ExtMap == nil {
		return e
	}
	return extMapEntry{
		DirEntry: e,
		name:     m.mapExtP(e.Name()),
	}
}

func (m Mirror) mapExtP(p string) string {
	if m.ExtMap == nil {
		return p
	}
	if ext := filepath.Ext(p); ext == "" {
		return p
	} else if mext, ok := m.ExtMap[ext]; ok {
		p = p[:len(p)-len(ext)] + mext
		return p
	}
	return p
}
