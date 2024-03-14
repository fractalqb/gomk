package gomkore

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

type BuildID = uint64

type Project struct {
	Dir string

	sync.Mutex

	parent    *Project
	goals     map[string]*Goal // TODO use key that respect Artefact type correctly
	actions   []*Action
	lastBuild BuildID
}

var _ Artefact = (*Project)(nil)

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

func (prj *Project) Actions() []*Action { return prj.actions }

func (prj *Project) FindGoal(name string) *Goal {
	return prj.goals[name]
}

func (prj *Project) Name(in *Project) string {
	if in == nil {
		return filepath.Base(prj.Dir)
	}
	n, _ := in.RelPath(prj.Dir)
	return n
}

func (prj *Project) String() string {
	tmp := prj.Dir
	if tmp == "" || tmp == "." {
		var err error
		if tmp, err = os.Getwd(); err != nil {
			return fmt.Sprintf("<error:%s>", err)
		}
	}
	return filepath.Base(tmp)
}

func (prj *Project) StateAt(in *Project) (time.Time, error) {
	if in == nil {
		in = prj
	}
	leafs := prj.Leafs()
	if len(leafs) == 0 {
		return time.Time{}, nil
	}
	t, err := leafs[0].Artefact.StateAt(in)
	if err != nil {
		return time.Time{}, err
	}
	for _, l := range leafs[1:] {
		if u, err := l.Artefact.StateAt(in); err != nil {
			return u, err
		} else if u.After(t) {
			t = u
		}
	}
	return t, nil
}

func (prj *Project) AbsPath(rel string) (string, error) {
	if filepath.IsAbs(prj.Dir) {
		return filepath.Join(prj.Dir, rel), nil
	}
	if prj.parent == nil {
		dir, err := filepath.Abs(prj.Dir)
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, rel), nil
	}
	pdir, err := prj.parent.AbsPath("")
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(pdir, prj.Dir, rel)), nil
}

func (prj *Project) RelPath(p string) (dir string, err error) {
	if p == "" || p == "." {
		return ".", nil
	}
	if !filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	if dir, err = prj.AbsPath(""); err != nil {
		return "", err
	}
	return filepath.Rel(dir, p)
}

func (prj *Project) RelPathTo(base, relTarget string) (string, error) {
	var err error
	if !filepath.IsAbs(base) {
		base, err = filepath.Abs(base)
		if err != nil {
			return "", err
		}
	}
	if relTarget, err = prj.AbsPath(relTarget); err != nil {
		return "", err
	}
	return filepath.Rel(base, relTarget)
}

func (prj *Project) Leafs() (ls []*Goal) {
	for _, g := range prj.goals {
		if len(g.PremiseOf()) == 0 {
			ls = append(ls, g)
		}
	}
	return ls
}

func (prj *Project) Roots() (rs []*Goal) {
	for _, g := range prj.goals {
		if len(g.ResultOf()) == 0 {
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
		p.premiseOf = append(p.premiseOf, a)
	}
	for _, r := range results {
		r.resultOf = append(r.resultOf, a)
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

func (prj *Project) Build() BuildID { return prj.lastBuild }

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
