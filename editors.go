package gomk

import (
	"io/fs"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

// ProjectEd is used with [Edit].
type ProjectEd struct{ p *Project }

func (ed ProjectEd) Project() *Project { return ed.p }

func (ed ProjectEd) NewAction(premises, results []GoalEd, op gomkore.Operation) ActionEd {
	prems, ress := goals(premises), goals(results)
	a, err := ed.p.NewAction(prems, ress, op)
	if err != nil {
		panic(err)
	}
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

func (ed ProjectEd) RelPath(p string) string {
	rp, err := ed.p.RelPath(p)
	if err != nil {
		panic(err)
	}
	return rp
}

func (ed ProjectEd) FsStat(a mkfs.Artefact) fs.FileInfo {
	return mustRet(mkfs.Stat(a, ed.p))
}

func (ed ProjectEd) FsExists(a mkfs.Artefact) bool {
	return mustRet(mkfs.Exists(a, ed.p))
}

// GoalEd is used with [Edit].
type GoalEd struct{ g *Goal }

func (ed GoalEd) Goal() *Goal { return ed.g }

func (ed GoalEd) Project() ProjectEd { return ProjectEd{ed.g.Project()} }

func (ed GoalEd) UpdateMode() gomkore.UpdateMode     { return ed.g.UpdateMode }
func (ed GoalEd) SetUpdateMode(m gomkore.UpdateMode) { ed.g.UpdateMode = m }

func (ed GoalEd) Artefact() gomkore.Artefact { return ed.g.Artefact }

func (ed GoalEd) IsAbstract() bool { return ed.g.IsAbstract() }

func (result GoalEd) By(op gomkore.Operation, premises ...GoalEd) GoalEd {
	prj := result.g.Project()
	prems := goals(premises)
	results := []*Goal{result.g}
	if _, err := prj.NewAction(prems, results, op); err != nil {
		panic(err)
	}
	return result
}

func (ed GoalEd) ImpliedBy(premises ...GoalEd) GoalEd {
	prj := ed.g.Project()
	if _, err := prj.NewAction(goals(premises), []*Goal{ed.g}, nil); err != nil {
		panic(err)
	}
	return ed
}

func goals(gs []GoalEd) []*Goal {
	var gls []*Goal
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

func (ed ActionEd) Action() *Action { return ed.a }

func (ed ActionEd) Project() ProjectEd {
	return ProjectEd{ed.a.Project()}
}

func (ed ActionEd) SetIgnoreError(ignore bool) {
	ed.a.IgnoreError = ignore
}
