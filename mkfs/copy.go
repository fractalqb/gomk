package mkfs

import (
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

// Copy [Operation] copies [Artefact] premises within the OS's filesystem to
// each of its results.
type Copy struct {
	MkDirMode fs.FileMode
}

var _ gomkore.Operation = Copy{}

func (Copy) Describe(*gomkore.Action, *gomkore.Env) string { return "FS copy" }

func (cp Copy) Do(tr *gomkore.Trace, a *gomkore.Action, _ *gomkore.Env) error {
	var prems []Artefact
	for _, pre := range a.Premises() {
		switch fsa := pre.Artefact.(type) {
		case gomkore.Abstract:
			// do nothing
		case Artefact:
			prems = append(prems, fsa)
		default:
			return fmt.Errorf("FS copy: illegal premise artefact type %T", pre)
		}
	}
	for _, res := range a.Results() {
		atf := res.Artefact
		if mrr, ok := atf.(Mirror); ok {
			atf = mrr.Dest
		}
		switch res := atf.(type) {
		case File:
			return cp.toFile(tr, a.Project(), res, prems)
		case DirList:
			return cp.toList(tr, a.Project(), res, prems)
		case DirTree:
			return cp.toTree(tr, a.Project(), res, prems)
		case gomkore.Abstract:
			// do nothing
		default:
			return fmt.Errorf("FS copy: illegal result artefact type %T", res)
		}
	}
	return nil
}

func (cp Copy) toFile(tr *gomkore.Trace, prj *gomkore.Project, dst File, srcs []Artefact) error {
	dstPath, err := prj.AbsPath(dst.Path())
	if err != nil {
		return err
	}
	if err := cp.provideDir(filepath.Dir(dstPath)); err != nil {
		return err
	}
	if len(srcs) == 1 {
		src, err := prj.AbsPath(srcs[0].Path())
		if err != nil {
			return err
		}
		st, err := os.Stat(src)
		if err != nil {
			return err
		}
		return cp.copyFile(tr, dstPath, src, st)
	}
	w, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("FsCopy to %s: %w", dst.Path(), err)
	}
	defer w.Close()
	for _, src := range srcs {
		srcPath, err := prj.AbsPath(src.Path())
		if err != nil {
			return err
		}
		if srcPath == dstPath {
			tr.Warn("FS copy: `source` to itself, skipping",
				`source`, src.Path(),
			)
			continue
		}
		tr.Debug("FS copy: append `src` -> `dst`",
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

// TODO respect dst.Filter
func (cp Copy) toList(tr *gomkore.Trace, prj *gomkore.Project, dst DirList, srcs []Artefact) error {
	dstPath, err := prj.AbsPath(dst.Path())
	if err != nil {
		return err
	}
	for _, src := range srcs {
		srcPath, err := prj.AbsPath(src.Path())
		if err != nil {
			return err
		}
		if strings.HasPrefix(srcPath, dstPath) {
			tr.Warn("FS copy: skipping `source` inside `target` directory",
				`source`, src.Path(),
				`target`, dst,
			)
		}
		switch src := src.(type) {
		case File:
			st, err := os.Stat(srcPath)
			if err != nil {
				return err
			}
			bnm := filepath.Base(src.Path())
			err = cp.copyFile(tr, filepath.Join(dstPath, bnm), srcPath, st)
			if err != nil {
				return err
			}
		case DirList:
			err = src.ls(srcPath, func(_ string, e fs.DirEntry) error {
				return cp.copyEntry(tr,
					filepath.Join(dstPath, e.Name()),
					filepath.Join(srcPath, e.Name()),
				)
			})
			if err != nil {
				return err
			}
		case DirTree:
			err = src.ls(srcPath, func(s string, e fs.DirEntry) error {
				return cp.copyEntry(tr,
					filepath.Join(dstPath, e.Name()),
					s,
				)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO respect dst.Filter
func (cp Copy) toTree(tr *gomkore.Trace, prj *gomkore.Project, dst DirTree, srcs []Artefact) error {
	dstPath, err := prj.AbsPath(dst.Path())
	if err != nil {
		return err
	}
	for _, src := range srcs {
		srcPath, err := copyCheckNesting(prj, dst, dstPath, src)
		if err != nil {
			tr.Warn("FS copy: `skipping`", `skipping`, err)
			continue
		}
		switch src := src.(type) {
		case File:
			st, err := os.Stat(srcPath)
			if err != nil {
				return err
			}
			bnm := filepath.Base(src.Path())
			err = cp.copyFile(tr, filepath.Join(dstPath, bnm), srcPath, st)
			if err != nil {
				return err
			}
		case DirList:
			err = src.ls(srcPath, func(_ string, e fs.DirEntry) error {
				return cp.copyEntry(tr,
					filepath.Join(dstPath, e.Name()),
					filepath.Join(srcPath, e.Name()),
				)
			})
			if err != nil {
				return err
			}
		case DirTree:
			err = src.ls(srcPath, func(s string, _ fs.DirEntry) error {
				return cp.copyEntry(tr,
					filepath.Join(dstPath, s),
					filepath.Join(srcPath, s),
				)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func copyCheckNesting(prj *gomkore.Project, dst Directory, dstPath string, src Artefact) (srcPath string, err error) {
	srcPath, err = prj.AbsPath(src.Path())
	if err != nil {
		return srcPath, err
	}
	if !strings.HasPrefix(dstPath, srcPath) {
		return srcPath, nil
	}
	switch src := src.(type) {
	case Directory:
		ok, err := src.Contains(prj, dst)
		if err != nil {
			return srcPath, err
		}
		if !ok {
			return srcPath, nil
		}
	case Mirror:
		ok, err := src.Dest.Contains(prj, dst)
		if err != nil {
			return srcPath, err
		}
		if !ok {
			return srcPath, nil
		}
	}
	return srcPath, fmt.Errorf("target '%s' inside source directory '%s'",
		dst.Path(),
		src.Path(),
	)
}

func (cp Copy) copyEntry(tr *gomkore.Trace, dst, src string) error {
	sstat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sstat.IsDir() {
		return cp.copyFile(tr, dst, src, sstat)
	}
	tr.Debug("FS copy: mkdir `src` -> `dst`",
		slog.String(`src`, src),
		slog.String(`dst`, dst),
	)
	return os.MkdirAll(dst, sstat.Mode().Perm())

}

func (cp Copy) copyFile(tr *gomkore.Trace, dst, src string, sstat fs.FileInfo) error {
	if src == dst {
		return nil
	}
	tr.Debug("FS copy: `src` -> `dst`",
		slog.String(`src`, src),
		slog.String(`dst`, dst),
	)
	if err := cp.provideDir(filepath.Dir(dst)); err != nil {
		return err
	}
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

func (cp Copy) provideDir(path string) error {
	if cp.MkDirMode == 0 {
		return nil
	}
	return os.MkdirAll(path, cp.MkDirMode)
}

func (cp Copy) WriteHash(h hash.Hash, a *gomkore.Action, _ *gomkore.Env) (bool, error) {
	for _, pre := range a.Premises() {
		fmt.Fprintln(h, pre.Name())
	}
	for _, res := range a.Results() {
		fmt.Fprintln(h, res.Name())
	}
	return true, nil
}
