package gomk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type FsCopy struct {
	MkDirs bool
}

func (cp FsCopy) BuildAction(prj *Project, premises, results []*Goal) (*Action, error) {
	check := func(g *Goal) error {
		switch g.Artefact.(type) {
		case File:
			return nil
		case Directory:
			return nil
		}
		return fmt.Errorf("FS copy: illegal artefact type %T", g.Artefact)
	}
	for _, p := range premises {
		if err := check(p); err != nil {
			return nil, err
		}
	}
	for _, r := range results {
		if err := check(r); err != nil {
			return nil, err
		}
	}
	return prj.NewAction(premises, results, cp), nil
}

func (FsCopy) Describe(*Project) string { return "FS copy" }

func (cp FsCopy) Do(_ context.Context, a *Action, env *Env) error {
	var prems []string
	for _, pre := range a.Premises {
		switch pre := pre.Artefact.(type) {
		case File:
			prems = append(prems, pre.In(a.Project()))
		case Directory:
			prems = append(prems, pre.In(a.Project()))
		default:
			return fmt.Errorf("FS copy: illegal premise artefact type %T", pre)
		}
	}
	for _, res := range a.Results {
		switch res := res.Artefact.(type) {
		case File:
			return cp.toFile(res.In(a.Project()), prems)
		case Directory:
			return cp.toDir(res.In(a.Project()), prems)
		default:
			return fmt.Errorf("FS copy: illegal result artefact type %T", res)
		}
	}
	return nil
}

func (cp FsCopy) toFile(dst string, srcs []string) error {
	if cp.MkDirs {
		os.MkdirAll(filepath.Dir(dst), 0777) // TODO Is umsak enough
	}
	w, err := os.Create(dst)
	if err != nil {
		return err // TODO context
	}
	defer w.Close()
	for _, src := range srcs {
		r, err := os.Open(src)
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

func (cp FsCopy) toDir(dst string, srcs []string) error {
	if cp.MkDirs {
		os.MkdirAll(dst, 0777) // TODO Is umsak enough
	}
	for _, src := range srcs {
		st, err := os.Stat(src)
		if err != nil {
			return err
		}
		if st.IsDir() {
			return errors.New("NYI: FS copy dir -> dir")
		} else {
			bnm := filepath.Base(src)
			if err = fsCopyFiles(filepath.Join(dst, bnm), src, st); err != nil {
				return err
			}
		}
	}
	return nil
}

func fsCopyFiles(dst, src string, sstat fs.FileInfo) error {
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
