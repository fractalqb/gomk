package gomk

import (
	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

func Convert(
	prems []GoalEd,
	result func(GoalEd) gomkore.Artefact,
	goalCfg func(GoalEd),
	op gomkore.Operation,
	actionCfg func(ActionEd),
) (gls []GoalEd) {
	for _, pre := range prems {
		atf := result(pre)
		if atf == nil {
			continue
		}
		prj := pre.Project()
		res := prj.Goal(atf)
		if goalCfg != nil {
			goalCfg(res)
		}
		_, act := res.By(op, pre)
		if actionCfg != nil {
			actionCfg(act)
		}
		gls = append(gls, res)
	}
	return gls
}

type ExtMap = map[string]string

type OutFile struct {
	Strip, Dest mkfs.Directory
	Ext         ExtMap
}

func (x OutFile) Artefact(g GoalEd) gomkore.Artefact {
	f, ok := g.Artefact().(mkfs.File)
	if !ok {
		return nil
	}
	if x.Dest != nil {
		f = mustRet(f.Moved(x.Strip, x.Dest))
	}
	if x.Ext == nil {
		return f
	}
	ext := f.Ext()
	if ext == "" {
		return f
	}
	if ext = x.Ext[ext]; ext == "" {
		return nil
	}
	return f.WithExt(ext)
}
