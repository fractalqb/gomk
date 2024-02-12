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
	if len(g.ResultOf) == 0 {
		return nil
	}
	for _, act := range g.ResultOf {
		for _, pre := range act.Premises {
			if err := bd.buildGoal(ctx, pre); err != nil {
				return err
			}
		}
	}
	var (
		updated bool
		err     error
	)
	switch g.UpdateMode.Actions() {
	case UpdAllActions:
		updated, err = bd.updateAll(ctx, g)
	case UpdSomeActions:
		updated, err = bd.updateSome(ctx, g)
	case UpdAnyAction:
		updated, err = bd.updateAny(ctx, g)
	case UpdOneAction:
		updated, err = bd.updateOne(ctx, g)
	default:
		err = fmt.Errorf("illegal update mode actions: %d", g.UpdateMode.Actions())
	}
	if err != nil {
		return err
	}
	if updated {
		g.stateAt = g.Artefact.StateAt()
		if ha, ok := g.Artefact.(HashableArtefact); ok {
			h := bd.newHash()
			if err := ha.StateHash(h); err != nil {
				return err
			}
			g.stateHash = h.Sum(nil)
		}
	}
	return nil
}

func (bd *Builder) updateAll(ctx context.Context, g *Goal) (bool, error) {
	chgs := bd.checkPreSates(g)
	if len(chgs) == 0 { // No changes
		return false, nil
	}
	for i, a := range g.ResultOf {
		if err := a.RunContext(ctx, bd.Env); err != nil {
			return i > 0, err // TODO really i>0 ???
		}
	}
	return true, nil
}

func (bd *Builder) updateSome(ctx context.Context, g *Goal) (bool, error) {
	chgs := bd.checkPreSates(g)
	if len(chgs) == 0 { // No changes
		return false, nil
	}
	for i, idx := range chgs {
		if err := g.ResultOf[idx].RunContext(ctx, bd.Env); err != nil {
			return i > 0, err // TODO really i>0 ???
		}
	}
	return true, nil

}

func (bd *Builder) updateAny(ctx context.Context, g *Goal) (bool, error) {
	chgs := bd.checkPreSates(g)
	if len(chgs) == 0 { // No changes
		return false, nil
	}
	if len(chgs) > 0 {
		err := g.ResultOf[chgs[0]].RunContext(ctx, bd.Env)
		return err == nil, err // TODO really err==nil ???
	}
	return true, nil
}

func (bd *Builder) updateOne(ctx context.Context, g *Goal) (bool, error) {
	chgs := bd.checkPreSates(g)
	switch len(chgs) {
	case 0:
		return false, nil
	case 1:
		err := g.ResultOf[chgs[0]].RunContext(ctx, bd.Env)
		return err == nil, err // TODO really err==nil ???
	}
	return false, fmt.Errorf("%d update actions for goal %s", len(chgs), g.Name())
}

func (bd *Builder) checkPreSates(g *Goal) (chgs []int) {
	for actIdx, act := range g.ResultOf {
		actTS := g.Artefact.StateAt()
		if actTS.IsZero() {
			chgs = append(chgs, actIdx)
			continue
		}
	PREMISE_LOOP:
		for _, pre := range act.Premises {
			switch {
			case pre.stateAt.IsZero():
				chgs = append(chgs, actIdx)
				break PREMISE_LOOP
			case actTS.Before(pre.stateAt):
				chgs = append(chgs, actIdx)
				break PREMISE_LOOP
			}
		}
	}
	return chgs
}

func (bd *Builder) newHash() hash.Hash {
	// TODO is there any cryptographic relevance in the use of this hash? If yes => Change!
	return md5.New()
}
