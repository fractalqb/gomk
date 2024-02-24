package gomk

import "git.fractalqb.de/fractalqb/gomk/gomkore"

func Convert(prems []GoalEd, result func(GoalEd) gomkore.Artefact, op gomkore.Operation) (gls []GoalEd) {
	for _, pre := range prems {
		atf := result(pre)
		if atf == nil {
			continue
		}
		prj := pre.Project()
		res := prj.Goal(atf)
		res.By(op, pre)
		gls = append(gls, res)
	}
	return gls
}
