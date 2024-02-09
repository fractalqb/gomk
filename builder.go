package gomk

import (
	"context"
	"fmt"
)

type Builder struct {
	Env    *Env
	LogDir string

	bid int64
}

// Project builds all leafs in prj.
func (bd *Builder) Project(prj *Project) error {
	return bd.ProjectContext(context.Background(), prj)
}

// ProjectContext builds all leafs in prj.
func (bd *Builder) ProjectContext(ctx context.Context, prj *Project) error {
	prj.buildLock.Lock()
	defer prj.buildLock.Unlock()
	bd.bid = prj.buildID()
	leafs := prj.Leafs()
	if bd.Env == nil {
		bd.Env = DefaultEnv()
	}
	bd.Env.Log = bd.Env.Log.With("project", prj.String())
	for _, leaf := range leafs {
		if err := bd.buildGoal(ctx, leaf); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) Goal(gs ...*Goal) error {
	return bd.GoalsContext(context.Background(), gs...)
}

func (bd *Builder) GoalsContext(ctx context.Context, gs ...*Goal) error {
	if len(gs) == 0 {
		return nil
	}
	var prj *Project
	defer func() {
		if prj != nil {
			prj.buildLock.Unlock()
		}
	}()
	if bd.Env == nil {
		bd.Env = DefaultEnv()
	}
	bd.Env.Log = bd.Env.Log.With("project", prj.String())
	for _, g := range gs {
		if p := g.Project(); p != prj {
			if prj != nil {
				prj.buildLock.Unlock()
			}
			prj = p
			prj.buildLock.Lock()
		}
		bd.bid = prj.buildID()
		if err := bd.buildGoal(ctx, g); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) NamedGoals(prj *Project, names ...string) error {
	return bd.NamedGoalsContext(context.Background(), prj, names...)
}

func (bd *Builder) NamedGoalsContext(ctx context.Context, prj *Project, names ...string) error {
	var gs []*Goal
	for _, n := range names {
		g := prj.FindGoal(n)
		if g == nil {
			return fmt.Errorf("no goal named '%s' in project '%s'", n, prj.String())
		}
		gs = append(gs, g)
	}
	return bd.GoalsContext(ctx, gs...)
}

func (bd *Builder) buildGoal(ctx context.Context, g *Goal) error {
	if g.lastBuild >= bd.bid {
		return nil
	}
	g.lastBuild = bd.bid
	for _, act := range g.ResultOf {
		for _, pre := range act.Premises {
			if err := bd.buildGoal(ctx, pre); err != nil {
				return err
			}
		}
	}
	return bd.updateGoal(ctx, g)
}

func (bd *Builder) updateGoal(ctx context.Context, g *Goal) error {
	bd.Env.Log.Info("update `goal` in `project`", `goal`, g.String())
	if len(g.ResultOf) == 0 {
		return nil
	}
	if g.UpdateMode.Is(AllActions) {
		for _, a := range g.ResultOf {
			if err := a.RunContext(ctx, bd.Env); err != nil {
				return err
			}
		}
		return nil
	} else {
		return g.ResultOf[0].RunContext(ctx, bd.Env)
	}
}
