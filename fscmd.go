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
)

type FsArtefact interface {
	HashableArtefact
	Path() string
	Rel(*Project) string
	Stat() (fs.FileInfo, error)
	Exists() bool
}

type Directory interface {
	FsArtefact
	List() ([]fs.DirEntry, error)
}

type DirList string

var _ Directory = DirList("")

func (d DirList) Rel(prj *Project) string { return prj.relPath(d.Path()) }

func (d DirList) Path() string { return string(d) }

func (d DirList) Stat() (fs.FileInfo, error) { return os.Stat(d.Path()) }

func (d DirList) Exists() bool {
	_, err := d.Stat()
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

func (d DirList) List() ([]fs.DirEntry, error) { return os.ReadDir(string(d)) }

func (d DirList) Name(prj *Project) string { return d.Rel(prj) }

func (d DirList) StateAt() time.Time {
	st, err := d.Stat()
	if err != nil || !st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

func (d DirList) StateHash(h hash.Hash) error {
	entries, err := os.ReadDir(string(d))
	if err != nil {
		return err
	}
	for _, e := range entries {
		io.WriteString(h, e.Name())
		h.Write([]byte{'\n'})
	}
	return nil
}

type DirContent string

var _ Directory = DirContent("")

func (d DirContent) Rel(prj *Project) string { return prj.relPath(d.Path()) }

func (d DirContent) Path() string { return string(d) }

func (d DirContent) Stat() (fs.FileInfo, error) { return os.Stat(d.Path()) }

func (d DirContent) Exists() bool {
	_, err := d.Stat()
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

func (d DirContent) List() ([]fs.DirEntry, error) { return os.ReadDir(string(d)) }

func (d DirContent) Name(in *Project) string { return d.Rel(in) }

func (d DirContent) StateAt() (t time.Time) {
	err := filepath.WalkDir(d.Path(), func(_ string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			return nil
		}
		if dinfo, err := d.Info(); err != nil {
			return err
		} else if mt := dinfo.ModTime(); t.Before(mt) {
			t = mt
		}
		return nil
	})
	if err != nil {
		return time.Time{}
	}
	return t
}

func (d DirContent) StateHash(h hash.Hash) error {
	return filepath.WalkDir(d.Path(), func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			return nil
		}
		return File(path).StateHash(h)
	})
}

type File string

var _ FsArtefact = File("")

func (f File) Rel(prj *Project) string { return prj.relPath(f.Path()) }

func (f File) Path() string { return string(f) }

func (f File) Stat() (fs.FileInfo, error) { return os.Stat(f.Path()) }

func (f File) Exists() bool {
	_, err := f.Stat()
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

func (f File) Name(prj *Project) string { return f.Rel(prj) }

func (f File) StateAt() time.Time {
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

func (f File) StateHash(h hash.Hash) error {
	r, err := os.Open(f.Path())
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return err
	}
	defer r.Close()
	_, err = io.Copy(h, r)
	return err
}

type FsCopy struct {
	MkDirMode fs.FileMode
}

var (
	_ ActionBuilder = FsCopy{}
	_ Operation     = FsCopy{}
)

func (cp FsCopy) BuildAction(prj *Project, premises, results []*Goal) (*Action, error) {
	check := func(g *Goal) (bool, error) {
		switch g.Artefact.(type) {
		case File:
			return true, nil
		case DirList:
			return false, nil
		}
		return false, fmt.Errorf("FS copy: illegal artefact type %T", g.Artefact)
	}
	resultFiles := false
	for _, r := range results {
		isFile, err := check(r)
		if err != nil {
			return nil, err
		}
		resultFiles = resultFiles || isFile
	}
	for _, p := range premises {
		isFile, err := check(p)
		if err != nil {
			return nil, err
		}
		if resultFiles && !isFile {
			return nil, errors.New("FS copy: cannot copy directory to file")
		}
	}
	return prj.NewAction(premises, results, cp), nil
}

func (FsCopy) Describe(*Project) string { return "FS copy" }

func (cp FsCopy) Do(_ context.Context, a *Action, env *Env) (err error) {
	defer func() {
		if err != nil {
			env.Log.Error(err.Error())
		}
	}()
	var prems []FsArtefact
	for _, pre := range a.Premises {
		if fsa, ok := pre.Artefact.(FsArtefact); ok {
			prems = append(prems, fsa)
		} else {
			return fmt.Errorf("FS copy: illegal premise artefact type %T", pre)
		}
	}
	for _, res := range a.Results {
		switch res := res.Artefact.(type) {
		case File:
			return cp.toFile(a.Project(), res, prems, env)
		case Directory:
			return cp.toDir(a.Project(), res, prems, env)
		default:
			return fmt.Errorf("FS copy: illegal result artefact type %T", res)
		}
	}
	return nil
}

func (cp FsCopy) toFile(prj *Project, dst File, srcs []FsArtefact, env *Env) error {
	dstPath := dst.Rel(prj)
	if cp.MkDirMode != 0 {
		os.MkdirAll(filepath.Dir(dstPath), cp.MkDirMode)
	}
	if len(srcs) == 1 {
		src := srcs[0]
		st, err := src.Stat()
		if err != nil {
			return err
		}
		fsCopyFile(dstPath, src.Rel(prj), st, env.Log)
	}
	w, err := os.Create(dstPath)
	if err != nil {
		return err // TODO context
	}
	defer w.Close()
	for _, src := range srcs {
		srcPath := src.Rel(prj)
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
	dstPath := dst.Rel(prj)
	if cp.MkDirMode != 0 {
		if err := os.MkdirAll(dstPath, cp.MkDirMode); err != nil {
			return err
		}
	}
	for _, src := range srcs {
		st, err := src.Stat()
		if err != nil {
			return err
		}
		if st.IsDir() {
			srcPath := src.Rel(prj)
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
			err = fsCopyFile(filepath.Join(dstPath, bnm), src.Rel(prj), st, env.Log)
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

func (cp FsCopy) WriteHash(_ hash.Hash, a *Action, _ *Env) error {
	// TODO FsCopy.WriteHash()
	return errors.New("NYI: FsCopy.WriteHash()")
}

type FsConverter struct {
	OutExt    string
	Converter ActionBuilder
}

func (s FsConverter) ext() string {
	if s.OutExt == "" {
		return ""
	}
	if s.OutExt[0] != '.' {
		return "." + s.OutExt
	}
	return s.OutExt
}

func FsConvert(prj *Project, dir Directory, glob string, steps ...FsConverter) error {
	if len(steps) == 0 {
		return nil
	}
	res := prj.Goal(dir)
	var outGoals []*Goal
	err := filepath.WalkDir(dir.Path(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ok, err := filepath.Match(glob, path); err != nil {
			return err
		} else if !ok {
			return nil
		}
		ext := filepath.Ext(path)
		base := path[:len(path)-len(ext)]
		var resGoal *Goal
		for _, step := range steps {
			resFile := File(base + step.ext())
			resGoal = prj.Goal(resFile).By(step.Converter, prj.Goal(File(path)))
			path = resFile.Path()
		}
		outGoals = append(outGoals, resGoal)
		return nil
	})
	if err != nil {
		return err
	}
	if len(outGoals) > 0 {
		res.ImpliedBy(outGoals...)
	}
	return nil
}
