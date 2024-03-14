package gomkore

import (
	"context"
)

type Changer struct {
	updater
}

func (chg *Changer) Goal(g *Goal, tr *Trace) error {
	return chg.GoalContext(context.Background(), g, tr)
}

func (chg *Changer) GoalContext(ctx context.Context, g *Goal, tr *Trace) error {
	g.Project().LockBuild()
	defer g.Project().Unlock()
	if chg.Env == nil {
		chg.Env = DefaultEnv(tr)
	}
	tr.Info("Check change of `goal`", `goal`, g)
	for _, act := range g.PremiseOf() {
		for _, res := range act.Results() {
			err := chg.update(tr, res)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (chg *Changer) update(t *Trace, g *Goal) error {
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
