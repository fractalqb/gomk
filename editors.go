package gomk

import (
	"io/fs"

	"git.fractalqb.de/fractalqb/eloc/must"
	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

// ProjectEd is used with [Edit].
type ProjectEd struct{ p *gomkore.Project }

func (ed ProjectEd) Project() *gomkore.Project { return ed.p }

func (ed ProjectEd) NewAction(premises, results []GoalEd, op gomkore.Operation) ActionEd {
	prems, ress := goals(premises), goals(results)
	a := must.Ret(ed.p.NewAction(prems, ress, op))
	return ActionEd{a}
}

func (ed ProjectEd) Dir() string { return ed.p.Dir }

func (ed ProjectEd) Goal(atf gomkore.Artefact) GoalEd {
	g, err := ed.p.Goal(atf)
	if err != nil {
		panic(err)
	}
	return GoalEd{g}
}

func (ed ProjectEd) AbstractGoal(name string) GoalEd {
	return ed.Goal(gomkore.Abstract(name))
}

func (ed ProjectEd) RelPath(p string) string {
	rp, err := ed.p.RelPath(p)
	if err != nil {
		panic(err)
	}
	return rp
}

func (ed ProjectEd) FsStat(a mkfs.Artefact) fs.FileInfo {
	return must.Ret(mkfs.Stat(a, ed.p))
}

func (ed ProjectEd) FsExists(a mkfs.Artefact) bool {
	return must.Ret(mkfs.Exists(a, ed.p))
}

// GoalEd is used with [Edit].
type GoalEd struct{ g *gomkore.Goal }

func GoalEds(prj ProjectEd, f gomkore.GoalFactory) (geds []GoalEd) {
	gs, err := f.Goals(prj.Project())
	if err != nil {
		panic(err)
	}
	geds = make([]GoalEd, len(gs))
	for i, g := range gs {
		geds[i] = GoalEd{g}
	}
	return geds
}

func (ed GoalEd) Goal() *gomkore.Goal { return ed.g }

func (ed GoalEd) Project() ProjectEd { return ProjectEd{ed.g.Project()} }

func (ed GoalEd) UpdateMode() gomkore.UpdateMode     { return ed.g.UpdateMode }
func (ed GoalEd) SetUpdateMode(m gomkore.UpdateMode) { ed.g.UpdateMode = m }

func (ed GoalEd) Removable() bool        { return ed.g.Removable }
func (ed GoalEd) SetRemovable(flag bool) { ed.g.Removable = flag }

func (ed GoalEd) Artefact() gomkore.Artefact { return ed.g.Artefact }

func (ed GoalEd) IsAbstract() bool { return ed.g.IsAbstract() }

func (result GoalEd) By(op gomkore.Operation, premises ...GoalEd) (GoalEd, ActionEd) {
	prj := result.g.Project()
	prems := goals(premises)
	results := []*gomkore.Goal{result.g}
	act, err := prj.NewAction(prems, results, op)
	if err != nil {
		panic(err)
	}
	return result, ActionEd{act}
}

func (ed GoalEd) ImpliedBy(premises ...GoalEd) GoalEd {
	prj := ed.g.Project()
	if _, err := prj.NewAction(goals(premises), []*gomkore.Goal{ed.g}, nil); err != nil {
		panic(err)
	}
	return ed
}

func goals(gs []GoalEd) []*gomkore.Goal {
	var gls []*gomkore.Goal
	if l := len(gs); l > 0 {
		gls = make([]*gomkore.Goal, l)
		for i, p := range gs {
			gls[i] = p.g
		}
	}
	return gls
}

// ActionEd is used with [Edit].
type ActionEd struct{ a *gomkore.Action }

func (ed ActionEd) Action() *gomkore.Action { return ed.a }

func (ed ActionEd) Project() ProjectEd {
	return ProjectEd{ed.a.Project()}
}

func (ed ActionEd) SetIgnoreError(ignore bool) {
	ed.a.IgnoreError = ignore
}
