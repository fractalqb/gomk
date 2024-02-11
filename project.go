package gomk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"
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
	pp := in.relPath(prj.Dir)
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

func (prj *Project) WriteDot(w io.Writer) (n int, err error) {
	defer func() {
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				err = p
			default:
				panic(p)
			}
		}
	}()
	akku := func(p int, err error) {
		n += p
		if err != nil {
			panic(err)
		}
	}
	akku(fmt.Fprintf(w, "digraph \"%s\" {\n", prj.Name(nil)))
	for n, g := range prj.goals {
		tn := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()
		var updMode string
		if len(g.ResultOf) > 1 {
			switch g.UpdateMode.Actions() {
			case UpdOneAction:
				updMode = " 1"
			case UpdSomeActions:
				updMode = " +"
			case UpdAllActions:
				updMode = " *"
			}
		}
		if _, ok := g.Artefact.(Abstract); ok {
			akku(fmt.Fprintf(w,
				"\t\"%p\" [shape=box,style=dashed,label=\"%s%s\"];\n",
				g,
				n,
				updMode,
			))
		} else {
			akku(fmt.Fprintf(w, "\t\"%p\" [shape=record,label=\"{%s%s|%s}\"];\n",
				g,
				tn,
				updMode,
				n,
			))
		}
		for _, a := range g.ResultOf {
			if a.Op == nil {
				akku(fmt.Fprintf(w, "\t\"%p\" [shape=none,label=\"implicit\"];\n", a))
			} else {
				akku(fmt.Fprintf(w, "\t\"%p\" [label=\"%s\"];\n", a, a.String()))
			}
			akku(fmt.Fprintf(w, "\t\"%p\" -> \"%p\";\n", a, g))
			for _, p := range a.Premises {
				akku(fmt.Fprintf(w, "\t\"%p\" -> \"%p\";\n", p, a))
			}
		}
	}
	akku(fmt.Fprintln(w, "}"))
	return
}

func (prj *Project) NewAction(premises, results []*Goal, op Operation) *Action {
	err := prj.consistentPrj(premises, results)
	if err != nil {
		op = badOp{op: op, err: err}
	} else if prj == nil {
		op = badOp{op: op, err: errors.New("no project")}
	}
	a := &Action{
		Premises: premises,
		Results:  results,
		Op:       op,
		prj:      prj,
	}
	for _, p := range premises {
		p.PremiseOf = append(p.PremiseOf, a)
	}
	for _, r := range results {
		r.ResultOf = append(r.ResultOf, a)
	}
	return a
}

func (prj *Project) consistentPrj(premises, results []*Goal) error {
	for _, g := range premises {
		if prj == nil {
			prj = g.Project()
		} else if p := g.Project(); p != prj {
			return fmt.Errorf("premise '%s' not in project '%s'",
				p.String(),
				prj.String(),
			)
		}
	}
	for _, g := range results {
		if prj == nil {
			prj = g.Project()
		} else if p := g.Project(); p != prj {
			return fmt.Errorf("result '%s' not in project '%s'",
				p.String(),
				prj.String(),
			)
		}
	}
	return nil
}

func (prj *Project) buildID() int64 {
	prj.lastBuild++
	return prj.lastBuild
}

func (prj *Project) relPath(s string) string {
	var (
		tmp string
		err error
	)
	if prj.Dir == "" {
		tmp, err = filepath.Rel(".", s)
	} else {
		tmp, err = filepath.Rel(prj.Dir, s)
	}
	if err != nil {
		return filepath.Clean(s)
	}
	return tmp
}
