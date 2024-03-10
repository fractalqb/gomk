package gomk

import (
	"context"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type Changer struct {
	updater
}

func (chg *Changer) Goal(g *Goal) error {
	return chg.GoalContext(context.Background(), g)
}

func (chg *Changer) GoalContext(ctx context.Context, g *Goal) error {
	bid := g.Project().LockBuild()
	defer g.Project().Unlock()
	if chg.Env == nil {
		chg.Env = DefaultEnv()
	}
	chg.Env.Log.Info("Check change of `goal`", `goal`, g)
	return chg.update(ctx, g, bid)
}

func (chg *Changer) update(ctx context.Context, g *Goal, bid gomkore.BuildID) error {
	if ok, err := chg.updateGoal(ctx, g); err != nil {
		return err
	} else if ok {
		for _, act := range g.PremiseOf() {
			for _, res := range act.Results() {
				if err := chg.update(ctx, res, bid); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
