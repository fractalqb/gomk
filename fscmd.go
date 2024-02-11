package gomk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"log/slog"
)

type FsCopy struct {
	MkDirs bool
}

func (cp FsCopy) BuildAction(prj *Project, premises, results []*Goal) (*Action, error) {
	check := func(g *Goal) (bool, error) {
		switch g.Artefact.(type) {
		case File:
			return true, nil
		case Directory:
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
			return cp.toFile(res.In(a.Project()), prems, env)
		case Directory:
			return cp.toDir(res.In(a.Project()), prems, env)
		default:
			return fmt.Errorf("FS copy: illegal result artefact type %T", res)
		}
	}
	return nil
}

func (cp FsCopy) toFile(dst string, srcs []string, env *Env) error {
	if cp.MkDirs {
		os.MkdirAll(filepath.Dir(dst), 0777) // TODO Is umsak enough
	}
	if len(srcs) == 1 {
		src := srcs[0]
		st, err := os.Stat(src)
		if err != nil {
			return err
		}
		fsCopyFile(dst, src, st, env.Log)
	}
	w, err := os.Create(dst)
	if err != nil {
		return err // TODO context
	}
	defer w.Close()
	for _, src := range srcs {
		env.Log.Debug("FS copy: append `src` -> `dst`",
			slog.String(`src`, src),
			slog.String(`dst`, dst),
		)
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

func (cp FsCopy) toDir(dst string, srcs []string, env *Env) error {
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
			err = fsCopyFile(filepath.Join(dst, bnm), src, st, env.Log)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func fsCopyFile(dst, src string, sstat fs.FileInfo, log *slog.Logger) error {
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
