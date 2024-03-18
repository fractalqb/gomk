package gomkore

import (
	"errors"
	"fmt"
	"hash"
	"time"
)

// TODO Add a "dry-run" option
type Builder struct {
	updater
}

var _ Operation = (*Builder)(nil)

func NewBuilder(tr *Trace, env *Env) (*Builder, error) {
	if tr == nil {
		return nil, errors.New("no trace for new builder")
	}
	return &Builder{
		updater: updater{
			trace: tr,
			env:   env,
		},
	}, nil
}

// Project builds all leafs in prj.
func (bd *Builder) Project(prj *Project) error {
	bd.bid = prj.LockBuild()
	defer prj.Unlock()
	if bd.env == nil {
		bd.env = DefaultEnv(bd.trace)
	}
	return bd.buildPrj(bd.trace, prj)
}

func (bd *Builder) Goals(gs ...*Goal) error {
	if len(gs) == 0 {
		return nil
	}
	var (
		prj      *Project
		prjStart time.Time
	)
	defer func() {
		if prj != nil {
			bd.trace.doneProject(prj, "building", time.Since(prjStart))
			prj.Unlock()
		}
	}()
	for _, g := range gs {
		if p := g.Project(); p != prj {
			if prj != nil {
				bd.trace.doneProject(prj, "building", time.Since(prjStart))
				prj.Unlock()
			}
			prj = p
			bd.trace.startProject(prj, "building")
			prjStart = time.Now()
			if bd.env == nil {
				bd.env = DefaultEnv(bd.trace)
			}
			prj.LockBuild()
		}
		if err := bd.buildGoal(bd.trace, g); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) NamedGoals(prj *Project, names ...string) error {
	var gs []*Goal
	for _, n := range names {
		g := prj.FindGoal(n)
		if g == nil {
			return fmt.Errorf("no goal named '%s' in project '%s'", n, prj.String())
		}
		gs = append(gs, g)
	}
	return bd.Goals(gs...)
}

func (bd *Builder) buildPrj(tr *Trace, prj *Project) error {
	start := time.Now()
	tr = tr.pushProject(prj)
	tr.startProject(prj, "building")
	leafs := prj.Leafs()
	for _, leaf := range leafs {
		if err := bd.buildGoal(tr, leaf); err != nil {
			return err
		}
	}
	tr.doneProject(prj, "building", time.Since(start))
	return nil
}

func (bd *Builder) buildGoal(tr *Trace, g *Goal) error {
	if g.LockBuild() == 0 {
		return nil
	}
	defer g.Unlock()

	tr = tr.pushGoal(g)
	tr.checkGoal(g)
	if len(g.ResultOf()) == 0 {
		return nil
	}
	for _, act := range g.ResultOf() {
		for _, pre := range act.Premises() {
			if err := bd.buildGoal(tr, pre); err != nil {
				return err
			}
		}
	}

	_, err := bd.updateGoal(tr, g)
	return err
}

func (bd *Builder) Describe(*Action, *Env) string {
	return "Build project" // TODO better description
}

func (bd *Builder) Do(tr *Trace, a *Action, env *Env) error {
	var prjs []*Project
	for _, res := range a.Results() {
		switch a := res.Artefact.(type) {
		case Abstract:
			continue
		case *Project:
			prjs = append(prjs, a)
		default:
			return fmt.Errorf("illegal project build target %T", res)
		}
	}
	for _, prj := range prjs {
		if err := bd.buildPrj(tr, prj); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) WriteHash(h hash.Hash, a *Action, env *Env) (bool, error) {
	return false, nil // TODO good idea for how to hash build project operation
}

type BuildTracer interface {
	TracerCommon

	RunAction(*Trace, *Action)
	RunImplicitAction(*Trace, *Action)

	ScheduleResTimeZero(t *Trace, a *Action, res *Goal)
	ScheduleNotPremises(t *Trace, a *Action, res *Goal)
	SchedulePreTimeZero(t *Trace, a *Action, res, pre *Goal)
	ScheduleOutdated(t *Trace, a *Action, res, pre *Goal)

	CheckGoal(t *Trace, g *Goal)
	GoalUpToDate(t *Trace, g *Goal)
	GoalNeedsActions(t *Trace, g *Goal, n int)
}
