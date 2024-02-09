package gomk

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"git.fractalqb.de/fractalqb/eloc"
	"git.fractalqb.de/fractalqb/eloc/must"
)

type Project struct {
	Dir string

	goals     map[string]*Goal
	buildLock sync.Mutex
	lastBuild int64
}

func NewProject(dir string) *Project {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	prj := &Project{
		Dir:   dir,
		goals: make(map[string]*Goal),
	}
	return prj
}

func (prj *Project) Goal(atf Artefact) *Goal {
	if atf == nil {
		n := fmt.Sprintf("artefact-%d", len(prj.goals))
		atf = Abstract(n)
	}
	name := atf.Name(prj)
	if g := prj.goals[name]; g != nil {
		return g
	}
	g := &Goal{
		Artefact: atf,
		prj:      prj,
	}
	prj.goals[name] = g
	return g
}

func (prj *Project) FindGoal(name string) *Goal {
	return prj.goals[name]
}

func (prj *Project) Name(in *Project) string {
	pp := prj.relPath(prj.Dir)
	if prj == nil {
		return filepath.Base(pp)
	}
	return pp
}

func (prj *Project) String() string {
	tmp := prj.Dir
	if tmp == "" || tmp == "." {
		tmp, _ = filepath.Abs(tmp) // TODO error
	}
	return filepath.Base(tmp)
}

func (prj *Project) StateAt() time.Time {
	leafs := prj.Leafs()
	if len(leafs) == 0 {
		return time.Time{}
	}
	t := leafs[0].Artefact.StateAt()
	for _, l := range leafs[1:] {
		u := l.Artefact.StateAt()
		if u.After(t) {
			t = u
		}
	}
	return t
}

func (prj *Project) Leafs() (ls []*Goal) {
	for _, g := range prj.goals {
		if len(g.PremiseOf) == 0 {
			ls = append(ls, g)
		}
	}
	return ls
}

func (prj *Project) Roots() (rs []*Goal) {
	for _, g := range prj.goals {
		if len(g.ResultOf) == 0 {
			rs = append(rs, g)
		}
	}
	return rs
}

func (prj *Project) WriteDot(w io.Writer) (err error) {
	eloc.RecoverAs(&err)
	must.Ret(fmt.Fprintf(w, "digraph \"%s\" {\n", prj.Name(nil)))
	for n, g := range prj.goals {
		tn := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()
		var updMode string
		if len(g.ResultOf) > 1 {
			switch g.UpdateMode & AllActions {
			case OneAction:
				updMode = " 1"
			case SomeActions:
				updMode = " +"
			case AllActions:
				updMode = " *"
			}
		}
		fmt.Fprintf(w, "\t\"%p\" [shape=record,label=\"{%s%s|%s}\"];\n", g, tn, updMode, n)
		for _, a := range g.ResultOf {
			if a.Op == nil {
				fmt.Fprintf(w, "\t\"%p\" [shape=none,label=\"implicit\"];\n", a)
			} else {
				fmt.Fprintf(w, "\t\"%p\" [label=\"%s\"];\n", a, a.String())
			}
			fmt.Fprintf(w, "\t\"%p\" -> \"%p\";\n", a, g)
			for _, p := range a.Premises {
				fmt.Fprintf(w, "\t\"%p\" -> \"%p\";\n", p, a)
			}
		}
	}
	must.Ret(fmt.Fprintln(w, "}"))
	return nil
}

func (prj *Project) buildID() int64 {
	prj.lastBuild++
	return prj.lastBuild
}

func (prj *Project) relPath(s string) string {
	if filepath.IsAbs(s) {
		return s
	}
	if prj == nil {
		if s == "" || s == "." {
			tmp, err := filepath.Abs(s)
			if err == nil {
				return tmp
			}
		}
		return s
	}
	r, err := filepath.Rel(prj.Dir, s)
	if err != nil {
		return filepath.Clean(s)
	}
	return r
}
