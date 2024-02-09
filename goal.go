package gomk

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"reflect"
	"time"
)

type UpdateMode uint

const (
	OneAction   UpdateMode = 1
	SomeActions UpdateMode = 2
	AllActions  UpdateMode = 3
	Concurrent  UpdateMode = 4
)

func (m UpdateMode) Is(test UpdateMode) bool {
	if t := test & 3; t != m&3 {
		return false
	}
	t := test & Concurrent
	return t == 0 || (m&Concurrent == Concurrent)
}

// A Goal is something you want to achieve in your [Project]. Each goal is
// associated with an [Artefact] – generally something tangible that is
// considered available and up-to-date when the goal is achieved. A special case
// is the [Abstract] artefact that simply provides a name for abstract goals.
// Goals can be achieved through actions. The goal is then a result of each
// [Action]. On the other hand, a goal can also be the premise for one or more
// actions. Such dependent actions may not be carried out before the goal is
// reached.
type Goal struct {
	UpdateMode UpdateMode
	ResultOf   []*Action // Actions that result in this goal.
	PremiseOf  []*Action // Dependent actions of this goal.
	Artefact   Artefact

	prj       *Project
	lastBuild int64
}

func (g *Goal) ByOp(op Operation, premises ...*Goal) *Goal {
	newAction(premises, []*Goal{g}, op)
	return g
}

func (g *Goal) ImplicitBy(premises ...*Goal) *Goal {
	newAction(premises, []*Goal{g}, nil)
	return g
}

func (g *Goal) ByAction(a ActionBuilder, premises ...*Goal) *Goal {
	_, err := a.NewAction(g.Project(), premises, []*Goal{g})
	if err != nil {
		return g.ByOp(badOp{err: err})
	}
	return g
}

func (g *Goal) Project() *Project { return g.prj }

func (g *Goal) Update(env *Env) error {
	return g.UpdateContext(context.Background(), env)
}

func (g *Goal) UpdateContext(ctx context.Context, env *Env) error {
	if env == nil {
		env = DefaultEnv()
	}
	env.Log.Info("update `goal` in `project`", `goal`, g.String())
	if len(g.ResultOf) == 0 {
		return nil
	}
	if g.UpdateMode.Is(AllActions) {
		for _, a := range g.ResultOf {
			if err := a.RunContext(ctx, env); err != nil {
				return err
			}
		}
		return nil
	} else {
		return g.ResultOf[0].RunContext(ctx, env)
	}
}

func (g *Goal) Name() string { return g.Artefact.Name(g.Project()) }

func (g *Goal) Valid() (*Goal, error) {
	var errs []error
	for _, a := range g.ResultOf {
		if err := a.Valid(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, a := range g.PremiseOf {
		if err := a.Valid(); err != nil {
			errs = append(errs, err)
		}
	}
	switch len(errs) {
	case 0:
		return g, nil
	case 1:
		return g, errs[1]
	}
	return g, errors.Join(errs...)
}

func (g *Goal) String() string {
	tn := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()
	an := g.Name()
	return fmt.Sprintf("[%s]%s", an, tn)
}

type Artefact interface {
	Name(in *Project) string
	StateAt() time.Time
}

type Abstract string

func (a Abstract) Name(*Project) string { return string(a) }

func (a Abstract) StateAt() time.Time { return time.Time{} }

type Directory string

func (d Directory) Path() string { return string(d) }

func (d Directory) Name(in *Project) string { return in.relPath(d.Path()) }

func (d Directory) StateAt() time.Time {
	st, err := d.Stat()
	if err != nil || !st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

func (d Directory) Stat() (fs.FileInfo, error) { return os.Stat(d.Path()) }

type HashableArtefact interface {
	Artefact
	WriteHash(hash.Hash) error
}

type File string

func (f File) Path() string { return string(f) }

func (f File) StateAt() time.Time {
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

func (f File) Stat() (fs.FileInfo, error) { return os.Stat(f.Path()) }

func (f File) Name(in *Project) string { return in.relPath(f.Path()) }

func (f File) WriteHash(h hash.Hash) error {
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
