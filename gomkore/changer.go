package gomkore

import "errors"

type Changer struct {
	updater
}

func NewChanger(tr *Trace, env *Env) (*Changer, error) {
	if tr == nil {
		return nil, errors.New("no trace for new changer")
	}
	return &Changer{
		updater: updater{
			trace: tr,
			env:   env,
		},
	}, nil
}

func (chg *Changer) Goals(gs ...*Goal) error {
	if len(gs) == 0 {
		return nil
	}
	var prj *Project
	defer func() {
		if prj != nil {
			chg.trace.doneProject(prj, "updating", 0) // TODO duration
			prj.Unlock()
		}
	}()
	for _, g := range gs {
		if p := g.Project(); p != prj {
			if prj != nil {
				chg.trace.doneProject(prj, "updating", 0) // TODO duration
				prj.Unlock()
			}
			prj = p
			chg.trace.startProject(prj, "updating")
			if chg.env == nil {
				chg.env = DefaultEnv(chg.trace)
			}
			prj.LockBuild()
		}
		chg.trace.checkGoal(g)
		for _, act := range g.PremiseOf() {
			for _, res := range act.Results() {
				err := chg.update(chg.trace, res)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (chg *Changer) update(t *Trace, g *Goal) error {
	t = t.pushGoal(g)
	if ok, err := chg.updateGoal(t, g); err != nil {
		return err
	} else if ok {
		for _, act := range g.PremiseOf() {
			for _, res := range act.Results() {
				if err := chg.update(t, res); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
