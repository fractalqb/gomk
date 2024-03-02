package gomk

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"log/slog"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type FsArtefact interface {
	gomkore.Artefact
	// Path retuns the OS specific path (see filepath.FromSlash)
	Path() string
	Rel(in ProjectEd) string
	Stat(in ProjectEd) (fs.FileInfo, error)
	Exists(in ProjectEd) bool
}

func FsStat(a FsArtefact, in *Project) (fs.FileInfo, error) {
	return os.Stat(in.RelPath(a.Path()))
}

func FsExists(a FsArtefact, in *Project) bool {
	_, err := FsStat(a, in)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

type DirListSelect int

const (
	DirListFiles = 1
	DirListDirs  = 2
)

func (sel DirListSelect) ok(e fs.DirEntry) bool {
	if sel == 0 {
		return true
	}
	if sel&DirListFiles == 0 && !e.IsDir() {
		return false
	}
	if sel&DirListDirs == 0 && e.IsDir() {
		return false
	}
	return true
}

type Directory interface {
	FsArtefact
	List(in ProjectEd) ([]string, error)
}

func FsList(d Directory, in *Project) (ls []string, err error) {
	switch d := d.(type) {
	case DirList:
		return FsDirList(d, in)
	case DirContent:
		return FsDirContent(d, in)
	}
	return nil, fmt.Errorf("unsupported directory type %T", d)
}

func FsGoals(prj ProjectEd, dir, dirTmpl Directory) (goals []GoalEd, err error) {
	ls, err := dir.List(prj)
	if err != nil {
		return nil, err
	}
	for _, e := range ls {
		pe := prj.p.RelPath(e)
		if st, err := os.Stat(pe); err != nil {
			return nil, err
		} else if st.IsDir() {
			switch dirTmpl := dirTmpl.(type) {
			case DirList:
				dir := dirTmpl
				dir.Dir = e
				g := prj.Goal(dir)
				if err != nil {
					return nil, err
				}
				goals = append(goals, g)
			case DirContent:
				dir := dirTmpl
				dir.Dir = e
				g := prj.Goal(dir)
				if err != nil {
					return nil, err
				}
				goals = append(goals, g)
			}
		} else {
			g := prj.Goal(File(e))
			goals = append(goals, g)
		}
	}
	return goals, nil
}

type DirList struct {
	Dir    string // UNIX stype path with '/'
	Glob   string
	Select DirListSelect
}

var _ Directory = DirList{}

func (d DirList) Rel(in ProjectEd) string { return in.RelPath(d.Path()) }

// Path implements [FsArtefact].Path
func (d DirList) Path() string { return filepath.FromSlash(d.Dir) }

func (d DirList) Stat(in ProjectEd) (fs.FileInfo, error) { return FsStat(d, in.p) }

func (d DirList) Exists(in ProjectEd) bool { return FsExists(d, in.p) }

func (d DirList) List(in ProjectEd) (ls []string, err error) {
	return FsDirList(d, in.p)
}

func FsDirList(d DirList, in *Project) (ls []string, err error) {
	prjDir := in.RelPath(d.Dir)
	rdir, err := os.ReadDir(prjDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range rdir {
		if !d.Select.ok(entry) {
			continue
		}
		if d.Glob != "" {
			ok, err := filepath.Match(d.Glob, entry.Name())
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
		}
		ls = append(ls, filepath.Join(prjDir, entry.Name()))
	}
	return ls, nil
}

func (d DirList) Name(prj *Project) string { return prj.RelPath(d.Dir) }

func (d DirList) StateAt(in *Project) time.Time {
	st, err := os.Stat(in.RelPath(d.Dir))
	if err != nil || !st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

type DirContent struct {
	Dir    string // UNIX stype path with '/'
	Glob   string
	Select DirListSelect
}

var _ Directory = DirContent{}

func (d DirContent) Rel(prj ProjectEd) string { return prj.RelPath(d.Path()) }

// Path implements [FsArtefact].Path
func (d DirContent) Path() string { return filepath.FromSlash(d.Dir) }

func (d DirContent) Stat(in ProjectEd) (fs.FileInfo, error) { return FsStat(d, in.p) }

func (d DirContent) Exists(in ProjectEd) bool { return FsExists(d, in.p) }

func (d DirContent) List(in ProjectEd) (ls []string, err error) {
	return FsDirContent(d, in.p)
}

func FsDirContent(d DirContent, in *Project) (ls []string, err error) {
	root := in.RelPath(d.Path())
	err = filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ok, err := d.ok(path, e, d.Select); err != nil {
			return err
		} else if ok {
			ls = append(ls, path)
		}
		return nil
	})
	return
}

func (d DirContent) Name(in *Project) string { return in.RelPath(d.Dir) }

func (d DirContent) StateAt(in *Project) (t time.Time) {
	root := in.RelPath(d.Dir)
	err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ok, err := d.ok(p, e, d.Select); err != nil {
			return err
		} else if !ok {
			return nil
		}
		if info, err := e.Info(); err != nil {
			return err
		} else if mt := info.ModTime(); t.Before(mt) {
			t = mt
		}
		return nil
	})
	if err != nil {
		return time.Time{}
	}
	return t
}

func (d DirContent) ok(p string, e fs.DirEntry, sel DirListSelect) (bool, error) {
	if !sel.ok(e) {
		return false, nil
	}
	if d.Glob == "" {
		return true, nil
	}
	return filepath.Match(d.Glob, filepath.Base(p))
}

type File string

var _ FsArtefact = File("")

func (f File) Rel(prj ProjectEd) string { return prj.RelPath(f.Path()) }

// Path implements [FsArtefact].Path
func (f File) Path() string { return filepath.FromSlash(string(f)) }

func (f File) Stat(in ProjectEd) (fs.FileInfo, error) { return FsStat(f, in.p) }

func (f File) Exists(in ProjectEd) bool { return FsExists(f, in.p) }

func (f File) Name(in *Project) string { return in.RelPath(f.Path()) }

func (f File) StateAt(in *Project) time.Time {
	st, err := os.Stat(in.RelPath(f.Path()))
	if err != nil || st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

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

type FileExt string

func (x FileExt) Convert(g GoalEd) gomkore.Artefact {
	if a, ok := g.g.Artefact.(File); ok {
		return a.WithExt(string(x))
	}
	return nil
}

type FsCopy struct {
	MkDirMode fs.FileMode
}

var _ gomkore.Operation = FsCopy{}

func (FsCopy) Describe(*Action, *Env) string { return "FS copy" }

func (cp FsCopy) Do(_ context.Context, a *Action, env *Env) (err error) {
	defer func() {
		if err != nil {
			env.Log.Error(err.Error())
		}
	}()
	var prems []FsArtefact
	for _, pre := range a.Premises() {
		switch fsa := pre.Artefact.(type) {
		case FsArtefact:
			prems = append(prems, fsa)
		case Abstract:
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
			return cp.toDir(a.Project(), res, prems, env)
		case Abstract:
			// do nothing
		default:
			return fmt.Errorf("FS copy: illegal result artefact type %T", res)
		}
	}
	return nil
}

func (cp FsCopy) toFile(prj *Project, dst File, srcs []FsArtefact, env *Env) error {
	dstPath := prj.RelPath(dst.Path())
	if cp.MkDirMode != 0 {
		os.MkdirAll(filepath.Dir(dstPath), cp.MkDirMode)
	}
	if len(srcs) == 1 {
		src := prj.RelPath(srcs[0].Path())
		st, err := os.Stat(src)
		if err != nil {
			return err
		}
		fsCopyFile(dstPath, src, st, env.Log)
	}
	w, err := os.Create(dstPath)
	if err != nil {
		return err // TODO context
	}
	defer w.Close()
	for _, src := range srcs {
		srcPath := prj.RelPath(src.Path())
		env.Log.Debug("FS copy: append `src` -> `dst`",
			slog.String(`src`, srcPath),
			slog.String(`dst`, dstPath),
		)
		r, err := os.Open(srcPath)
		if err != nil {
			return err // TODO context
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

func (cp FsCopy) toDir(prj *Project, dst Directory, srcs []FsArtefact, env *Env) error {
	dstPath := prj.RelPath(dst.Path())
	if cp.MkDirMode != 0 {
		if err := os.MkdirAll(dstPath, cp.MkDirMode); err != nil {
			return err
		}
	}
	for _, src := range srcs {
		srcPath := prj.RelPath(src.Path())
		st, err := os.Stat(srcPath)
		if err != nil {
			return err
		}
		if st.IsDir() {
			switch src := src.(type) {
			case DirContent:
				return sfCopyDir(dstPath, srcPath, env.Log)
			case DirList:
				srcBase := filepath.Base(srcPath)
				dstPath = filepath.Join(srcBase)
				if cp.MkDirMode != 0 {
					if err := os.Mkdir(dstPath, cp.MkDirMode); err != nil {
						return err
					}
				}
				return sfCopyDir(dstPath, srcPath, env.Log)
			default:
				return fmt.Errorf("FS IsDir = true for %T", src)
			}
		} else {
			bnm := filepath.Base(src.Path())
			err = fsCopyFile(filepath.Join(dstPath, bnm), srcPath, st, env.Log)
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

func (cp FsCopy) WriteHash(h hash.Hash, a *Action, _ *Env) (bool, error) {
	for _, pre := range a.Premises() {
		fmt.Fprintln(h, pre.Name())
	}
	for _, res := range a.Results() {
		fmt.Fprintln(h, res.Name())
	}
	return true, nil
}
