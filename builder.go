package gomk

import (
	"context"
	"crypto/md5"
	"fmt"
	"hash"
	"time"
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
	start := time.Now()
	prj.buildLock.Lock()
	defer prj.buildLock.Unlock()

	bd.bid = prj.buildID()
	if bd.Env == nil {
		bd.Env = DefaultEnv()
	}
	bd.Env.Log = bd.Env.Log.With("project", prj.String(), "build", bd.bid)
	bd.Env.Log.Info("`build` `project` in `dir`", `dir`, prj.Dir)
	leafs := prj.Leafs()
	for _, leaf := range leafs {
		if err := bd.buildGoal(ctx, leaf); err != nil {
			return err
		}
	}
	bd.Env.Log.Info("`build` of `project` `took`", `took`, time.Since(start))
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
	bd.Env.Log.Info("`build` `goal` in `project`", `goal`, g.String())
	for _, act := range g.ResultOf {
		for _, pre := range act.Premises {
			if err := bd.buildGoal(ctx, pre); err != nil {
				return err
			}
		}
	}
	if bd.needsUpdate(g) {
		return bd.updateGoal(ctx, g)
	}
	return nil
}

func (bd *Builder) needsUpdate(g *Goal) bool {
	gt := g.Artefact.StateAt()
	if gt.IsZero() {
		return true
	}
	for _, a := range g.ResultOf {
		for _, pre := range a.Premises {
			pt := pre.Artefact.StateAt()
			if pt.IsZero() || gt.Before(pt) {
				return true
			}
		}
	}
	return false
}

func (bd *Builder) updateGoal(ctx context.Context, g *Goal) error {
	if len(g.ResultOf) > 0 {
		if g.UpdateMode.Actions() == UpdAllActions {
			for _, a := range g.ResultOf {
				if err := a.RunContext(ctx, bd.Env); err != nil {
					return err
				}
			}
		} else if err := g.ResultOf[0].RunContext(ctx, bd.Env); err != nil {
			return err
		}
	}
	g.stateAt = g.Artefact.StateAt()
	if ha, ok := g.Artefact.(HashableArtefact); ok {
		h := bd.newHash()
		if err := ha.WriteHash(h); err != nil {
			return err
		}
		g.stateHash = h.Sum(nil)
	}
	return nil
}

func (bd *Builder) newHash() hash.Hash {
	// TODO is there any cryptographic relevance in the use of this hash? If yes => Change!
	return md5.New()
}
