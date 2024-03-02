package gomkore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"
)

type BuildID = uint64

type Project struct {
	Dir string

	sync.Mutex

	parent    *Project
	goals     map[string]*Goal
	actions   []*Action
	lastBuild BuildID
}

var _ Artefact = (*Project)(nil)

// TODO _ Operation = (*Project)(nil) to support nested projects??? => ~Builder?

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

func (prj *Project) Goal(atf Artefact) (*Goal, error) {
	if atf == nil {
		n := fmt.Sprintf("artefact-%d", len(prj.goals))
		atf = Abstract(n)
	}
	name := atf.Name(prj)
	if g := prj.goals[name]; g != nil {
		return g, nil
	}
	if sub, ok := atf.(*Project); ok {
		if sub.parent != nil {
			return nil, fmt.Errorf("adding sub-project %s of %s to project %s",
				sub.Name(sub.parent),
				sub.parent.String(),
				prj.String(),
			)
		}
		sub.parent = prj
	}
	g := &Goal{
		Artefact: atf,
		prj:      prj,
	}
	prj.goals[name] = g
	return g, nil
}

// TODO interator as soon as it is available
func (prj *Project) Goals(addTo []*Goal) []*Goal {
	if len(prj.goals) == 0 {
		return nil
	}
	addTo = slices.Grow(addTo, len(prj.goals))
	for _, g := range prj.goals {
		addTo = append(addTo, g)
	}
	return addTo
}

func (prj *Project) FindGoal(name string) *Goal {
	return prj.goals[name]
}

func (prj *Project) Name(in *Project) string {
	if in == nil {
		return filepath.Base(prj.Dir)
	}
	return in.RelPath(prj.Dir)
}

func (prj *Project) String() string {
	tmp := prj.Dir
	if tmp == "" || tmp == "." {
		tmp, _ = filepath.Abs(tmp) // TODO error
	}
	return filepath.Base(tmp)
}

func (prj *Project) StateAt(in *Project) time.Time {
	if in == nil {
		in = prj
	}
	leafs := prj.Leafs()
	if len(leafs) == 0 {
		return time.Time{}
	}
	t := leafs[0].Artefact.StateAt(in)
	for _, l := range leafs[1:] {
		u := l.Artefact.StateAt(in)
		if u.After(t) {
			t = u
		}
	}
	return t
}

func (prj *Project) RelPath(p string) string {
	dir := prj.Dir
	if prj.parent != nil {
		dir = prj.parent.RelPath(dir)
	}
	var (
		tmp string
		err error
	)
	if dir == "" {
		tmp, err = filepath.Rel(".", p)
	} else {
		tmp, err = filepath.Rel(dir, p)
	}
	if err != nil {
		return filepath.Clean(p)
	}
	return tmp
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

// NewAction creates a new [Action] in project prj. There must be at least one
// result. All premises and results must belong to the same project prj.
func (prj *Project) NewAction(premises, results []*Goal, op Operation) (*Action, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("creating action %s without result",
			op.Describe(nil, nil),
		)
	}
	if err := prj.consistentPrj(premises, results); err != nil {
		return nil, err
	}
	a := &Action{
		Op:       op,
		prj:      prj,
		premises: premises,
		results:  results,
	}
	for _, p := range premises {
		p.PremiseOf = append(p.PremiseOf, a)
	}
	for _, r := range results {
		r.ResultOf = append(r.ResultOf, a)
	}
	if err := updateConsistency(results); err != nil {
		return nil, err
	}
	prj.actions = append(prj.actions, a)
	return a, nil
}

func (prj *Project) LockBuild() BuildID {
	prj.Lock()
	prj.lastBuild++
	return prj.lastBuild
}

func escDotID(id string) string {
	return strings.ReplaceAll(id, "\"", "\\\"")
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
	akku(fmt.Fprintf(w, "digraph \"%s\" {\n\trankdir=\"LR\"\n", escDotID(prj.Name(nil))))
	for n, g := range prj.goals {
		tn := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()
		var updMode string
		if len(g.ResultOf) > 1 {
			switch g.UpdateMode.Actions() {
			case UpdOneAction:
				updMode = " 1"
			case UpdAnyAction:
				updMode = " ?"
			case UpdSomeActions:
				updMode = " +"
			case UpdAllActions:
				updMode = " *"
			}
		}
		var style string
		if _, ok := g.Artefact.(Abstract); ok {
			if len(g.ResultOf) == 0 || len(g.PremiseOf) == 0 {
				style = ",style=\"dashed,bold\""
			} else {
				style = ",style=dashed"
			}
		} else if len(g.ResultOf) == 0 || len(g.PremiseOf) == 0 {
			style = ",style=bold"
		}
		akku(fmt.Fprintf(w, "\t\"%p\" [shape=record%s,label=\"{%s%s|%s}\"];\n",
			g,
			style,
			tn,
			updMode,
			escDotID(n),
		))
		for i, a := range g.ResultOf {
			if a.Op == nil {
				akku(fmt.Fprintf(w, "\t\"%p\" [shape=none,label=\"implicit\"];\n",
					a,
				))
			} else if len(a.premises) == 0 {
				akku(fmt.Fprintf(w,
					"\t\"%p\" [shape=box,style=\"rounded,bold\",label=\"%s\"];\n",
					a,
					escDotID(a.String()),
				))
			} else {
				akku(fmt.Fprintf(w,
					"\t\"%p\" [shape=box,style=rounded,label=\"%s\"];\n",
					a,
					escDotID(a.String()),
				))
			}
			var lb string
			if g.UpdateMode.Ordered() {
				lb = fmt.Sprintf(" [label=%d]", i+1)
			}
			akku(fmt.Fprintf(w, "\t\"%p\" -> \"%p\"%s;\n", a, g, lb))
		}
	}
	for _, act := range prj.actions {
		for _, p := range act.premises {
			akku(fmt.Fprintf(w, "\t\"%p\" -> \"%p\";\n", p, act))
		}
	}
	akku(fmt.Fprintln(w, "}"))
	return
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
