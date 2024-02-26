package gomk

import "git.fractalqb.de/fractalqb/gomk/gomkore"

type ProjectEd struct{ p *Project }

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

func (ed ProjectEd) RelPath(p string) string { return ed.p.RelPath(p) }

type GoalEd struct{ g *Goal }

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

type ActionEd struct{ a *gomkore.Action }

func (ed ActionEd) Project() ProjectEd {
	return ProjectEd{ed.a.Project()}
}

func (ed ActionEd) SetIgnoreError(ignore bool) {
	ed.a.IgnoreError = ignore
}
