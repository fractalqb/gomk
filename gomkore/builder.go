package gomkore

import (
	"context"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"time"
)

// TODO Add a "dry-run" option
type Builder struct {
	updater
}

var _ Operation = (*Builder)(nil)

// Project builds all leafs in prj.
func (bd *Builder) Project(prj *Project, tr *Trace) error {
	return bd.ProjectContext(context.Background(), prj, tr)
}

// ProjectContext builds all leafs in prj.
func (bd *Builder) ProjectContext(ctx context.Context, prj *Project, tr *Trace) error {
	bd.bid = prj.LockBuild()
	defer prj.Unlock()
	if bd.Env == nil {
		bd.Env = DefaultEnv(tr)
	}
	oldEnv, err := bd.prepLogDir()
	if err != nil {
		return err
	}
	defer func() { bd.restoreEnv(oldEnv) }()
	return bd.project(tr, prj)
}

func (bd *Builder) project(tr *Trace, prj *Project) error {
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

func (bd *Builder) Goals(tr *Trace, gs ...*Goal) error {
	return bd.GoalsContext(context.Background(), tr, gs...)
}

// TODO LogDir
func (bd *Builder) GoalsContext(ctx context.Context, tr *Trace, gs ...*Goal) error {
	if len(gs) == 0 {
		return nil
	}
	var prj *Project
	defer func() {
		if prj != nil {
			prj.Unlock()
		}
	}()
	for _, g := range gs {
		if p := g.Project(); p != prj {
			if prj != nil {
				tr.doneProject(prj, "building", 0) // TODO duration
				prj.Unlock()
			}
			tr.startProject(prj, "building")
			if bd.Env == nil {
				bd.Env = DefaultEnv(tr)
			}
			p.LockBuild()
			prj = p
		}
		if err := bd.buildGoal(tr, g); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) NamedGoals(prj *Project, tr *Trace, names ...string) error {
	return bd.NamedGoalsContext(context.Background(), tr, prj, names...)
}

func (bd *Builder) NamedGoalsContext(ctx context.Context, tr *Trace, prj *Project, names ...string) error {
	var gs []*Goal
	for _, n := range names {
		g := prj.FindGoal(n)
		if g == nil {
			return fmt.Errorf("no goal named '%s' in project '%s'", n, prj.String())
		}
		gs = append(gs, g)
	}
	return bd.GoalsContext(ctx, tr, gs...)
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

func (bd *Builder) restoreEnv(old *Env) {
	if bd.Env == old {
		return
	}
	if c, ok := bd.Env.Out.(io.Closer); ok {
		c.Close()
	}
	if c, ok := bd.Env.Err.(io.Closer); ok {
		c.Close()
	}
	bd.Env = old
}

func (bd *Builder) prepLogDir() (restore *Env, err error) {
	if bd.LogDir == "" {
		return bd.Env, nil
	}
	bdir := fmt.Sprintf("%s.%d", time.Now().Format("060102-150405"), bd.bid)
	bdir = filepath.Join(bd.LogDir, bdir)
	if bd.MkDirMode != 0 {
		if err = os.MkdirAll(bdir, bd.MkDirMode); err != nil {
			return bd.Env, err
		}
	} else if err = os.Mkdir(bdir, 0777); err != nil {
		return bd.Env, err
	}
	outFile, err := os.Create(filepath.Join(bdir, "build.out"))
	if err != nil {
		return bd.Env, err
	}
	errFile, err := os.Create(filepath.Join(bdir, "build.err"))
	if err != nil {
		outFile.Close()
		return bd.Env, err
	}
	restore = bd.Env
	bd.Env = restore.Clone()
	bd.Env.Out = stackedWriter{top: outFile, tail: restore.Out}
	bd.Env.Err = stackedWriter{top: errFile, tail: restore.Err}
	return restore, nil
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
		if err := bd.project(tr, prj); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) WriteHash(h hash.Hash, a *Action, env *Env) (bool, error) {
	return false, nil // TODO good idea to hash build project operation
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
