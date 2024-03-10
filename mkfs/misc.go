package mkfs

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type Artefact interface {
	gomkore.Artefact
	Path() string
}

func Stat(a Artefact, in *gomkore.Project) (fs.FileInfo, error) {
	p, err := in.RelPath(a.Path())
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
	case DirPath:
		return a.Moved(strip, dest)
	}
	return a, fmt.Errorf("cannot move FsArtefact %T", a)
}

type Directory interface {
	Artefact
	List(in *gomkore.Project) ([]string, error)
}

type FsCopy struct {
	MkDirMode fs.FileMode
}

var _ gomkore.Operation = FsCopy{}

func (FsCopy) Describe(*gomkore.Action, *gomkore.Env) string { return "FS copy" }

func (cp FsCopy) Do(_ context.Context, a *gomkore.Action, env *gomkore.Env) (err error) {
	defer func() {
		if err != nil {
			env.Log.Error(err.Error())
		}
	}()
	var prems []Artefact
	for _, pre := range a.Premises() {
		switch fsa := pre.Artefact.(type) {
		case Artefact:
			prems = append(prems, fsa)
		case gomkore.Abstract:
			// do nothing
		default:
			return fmt.Errorf("FS copy: illegal premise artefact type %T", pre)
		}
	}
	for _, res := range a.Results() {
		switch res := res.Artefact.(type) {
		case File:
			return cp.toFile(a.Project(), res, prems, env)
		case Directory:
			return cp.toDir(a.Project(), res.Path(), prems, env)
		case DirPath:
			return cp.toDir(a.Project(), res.Path(), prems, env)
		case gomkore.Abstract:
			// do nothing
		default:
			return fmt.Errorf("FS copy: illegal result artefact type %T", res)
		}
	}
	return nil
}

func (cp FsCopy) toFile(prj *gomkore.Project, dst File, srcs []Artefact, env *gomkore.Env) error {
	dstPath, err := prj.RelPath(dst.Path())
	if err != nil {
		return err
	}
	if cp.MkDirMode != 0 {
		os.MkdirAll(filepath.Dir(dstPath), cp.MkDirMode)
	}
	if len(srcs) == 1 {
		src, err := prj.RelPath(srcs[0].Path())
		if err != nil {
			return err
		}
		st, err := os.Stat(src)
		if err != nil {
			return err
		}
		fsCopyFile(dstPath, src, st, env.Log)
	}
	w, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("FsCopy to %s: %w", dst.Path(), err)
	}
	defer w.Close()
	for _, src := range srcs {
		srcPath, err := prj.RelPath(src.Path())
		if err != nil {
			return err
		}
		env.Log.Debug("FS copy: append `src` -> `dst`",
			slog.String(`src`, srcPath),
			slog.String(`dst`, dstPath),
		)
		r, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("FsCopy to %s: %w", dst.Path(), err)
		}
		_, err = io.Copy(w, r)
		if e := r.Close(); e != nil {
			if err == nil {
				return e
			}
			return errors.Join(err, e)
		}
	}
	return nil
}

func (cp FsCopy) toDir(prj *gomkore.Project, dst string, srcs []Artefact, env *gomkore.Env) error {
	dst, err := prj.RelPath(dst) // TODO should be AbsPath
	if err != nil {
		return err
	}
	if cp.MkDirMode != 0 {
		if err := os.MkdirAll(dst, cp.MkDirMode); err != nil {
			return err
		}
	}
	for _, src := range srcs {
		srcPath, err := prj.RelPath(src.Path())
		if err != nil {
			return err
		}
		st, err := os.Stat(srcPath)
		if err != nil {
			return err
		}
		if st.IsDir() {
			switch src := src.(type) {
			case DirTree:
				return sfCopyDir(dst, srcPath, env.Log)
			case DirList:
				srcBase := filepath.Base(srcPath)
				dst = filepath.Join(srcBase)
				if cp.MkDirMode != 0 {
					if err := os.Mkdir(dst, cp.MkDirMode); err != nil {
						return err
					}
				}
				return sfCopyDir(dst, srcPath, env.Log)
			default:
				return fmt.Errorf("FS IsDir = true for %T", src)
			}
		} else {
			bnm := filepath.Base(src.Path())
			err = fsCopyFile(filepath.Join(dst, bnm), srcPath, st, env.Log)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func sfCopyDir(dst, src string, log *slog.Logger) error {
	if src == dst {
		return nil
	}
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, _ error) error {
		if path == src {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dpath := filepath.Join(dst, rel)
		if d.IsDir() {
			if err := os.Mkdir(dpath, 0777); err != nil {
				return err
			}
		} else if stat, err := d.Info(); err != nil {
			return err
		} else if err := fsCopyFile(dpath, path, stat, log); err != nil {
			return err
		}
		return nil
	})
	return err
}

func fsCopyFile(dst, src string, sstat fs.FileInfo, log *slog.Logger) error {
	if src == dst {
		return nil
	}
	log.Debug("FS copy: `src` -> `dst`",
		slog.String(`src`, src),
		slog.String(`dst`, dst),
	)
	w, err := os.OpenFile(dst,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		sstat.Mode().Perm(),
	)
	if err != nil {
		return err
	}
	defer w.Close()
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()
	if _, err = io.Copy(w, r); err != nil {
		return err
	}
	return err
}

func (cp FsCopy) WriteHash(h hash.Hash, a *gomkore.Action, _ *gomkore.Env) (bool, error) {
	for _, pre := range a.Premises() {
		fmt.Fprintln(h, pre.Name())
	}
	for _, res := range a.Results() {
		fmt.Fprintln(h, res.Name())
	}
	return true, nil
}

func fsMove(path, strip, dest string) (string, error) {
	if strip != "" {
		var err error
		if path, err = filepath.Rel(strip, path); err != nil {
			return "", err
		}
	}
	return filepath.Join(dest, path), nil
}
