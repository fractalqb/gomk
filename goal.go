package gomk

import (
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

func (g *Goal) By(a ActionBuilder, premises ...*Goal) *Goal {
	_, err := a.BuildAction(g.Project(), premises, []*Goal{g})
	if err != nil {
		g.Project().NewAction(premises, []*Goal{g}, badOp{err: err})
		return g
	}
	return g
}

func (g *Goal) ImpliedBy(premises ...*Goal) *Goal {
	g.Project().NewAction(premises, []*Goal{g}, nil)
	return g
}

func (g *Goal) Project() *Project { return g.prj }

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

// Implement [Artefact]
func (d Directory) Name(prj *Project) string { return d.In(prj) }

// Implement [Artefact]
func (d Directory) StateAt() time.Time {
	st, err := d.Stat()
	if err != nil || !st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

func (d Directory) In(prj *Project) string { return prj.relPath(d.Path()) }

func (d Directory) Path() string { return string(d) }

func (d Directory) Stat() (fs.FileInfo, error) { return os.Stat(d.Path()) }

type HashableArtefact interface {
	Artefact
	WriteHash(hash.Hash) error
}

type File string

// Implement [Artefact]
func (f File) StateAt() time.Time {
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

// Implement [Artefact]
func (f File) Name(prj *Project) string { return f.In(prj) }

func (f File) In(prj *Project) string { return prj.relPath(f.Path()) }

func (f File) Path() string { return string(f) }

func (f File) Stat() (fs.FileInfo, error) { return os.Stat(f.Path()) }

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
