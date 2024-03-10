package gomk

import (
	"context"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/qblog"
)

// TODO Add a "dry-run" option
type Builder struct {
	updater
}

var _ gomkore.Operation = (*Builder)(nil)

// Project builds all leafs in prj.
func (bd *Builder) Project(prj *Project) error {
	return bd.ProjectContext(context.Background(), prj)
}

// ProjectContext builds all leafs in prj.
func (bd *Builder) ProjectContext(ctx context.Context, prj *Project) error {
	start := time.Now()
	bd.bid = prj.LockBuild()
	defer prj.Unlock()
	if bd.Env == nil {
		bd.Env = DefaultEnv()
	}
	oldEnv, err := bd.prepLogDir()
	if err != nil {
		return err
	}
	defer func() { bd.restoreEnv(oldEnv) }()
	tr := startTrace(bd.bid)
	bd.Env.Log.Info("`build` `project` in `dir`",
		`build`, fmt.Sprintf("@%d", bd.bid),
		`project`, prj.String(),
		`dir`, prj.Dir,
	)
	leafs := prj.Leafs()
	bd.goalIDSeq = 0
	for _, leaf := range leafs {
		if err := bd.buildGoal(ctx, leaf, tr); err != nil {
			return err
		}
	}
	bd.Env.Log.Info("`build` of `project` `took`",
		`build`, fmt.Sprintf("@%d", bd.bid),
		`project`, prj.String(),
		`took`, time.Since(start),
	)
	return nil
}

func (bd *Builder) Goals(gs ...*Goal) error {
	return bd.GoalsContext(context.Background(), gs...)
}

// TODO LogDir
func (bd *Builder) GoalsContext(ctx context.Context, gs ...*Goal) error {
	if len(gs) == 0 {
		return nil
	}
	var prj *Project
	defer func() {
		if prj != nil {
			prj.Unlock()
		}
	}()
	if bd.Env == nil {
		bd.Env = DefaultEnv()
	}
	origLog := bd.Env.Log
	var tr *trace
	bd.goalIDSeq = 0
	for _, g := range gs {
		if p := g.Project(); p != prj {
			if prj != nil {
				prj.Unlock()
			}
			bid := p.LockBuild()
			tr = startTrace(bid)
			prj = p
			bd.Env.Log = origLog.With("project", prj.String())
		}
		if err := bd.buildGoal(ctx, g, tr); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) GoalEds(gs ...GoalEd) error {
	return bd.GoalEdsContext(context.Background(), gs...)
}

func (bd *Builder) GoalEdsContext(ctx context.Context, gs ...GoalEd) error {
	tmp := make([]*gomkore.Goal, len(gs))
	for i, g := range gs {
		tmp[i] = g.g
	}
	return bd.GoalsContext(ctx, tmp...)
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

func (bd *Builder) buildGoal(ctx context.Context, g *Goal, tr *trace) error {
	if g.LockBuild() == 0 {
		return nil
	}
	defer g.Unlock()

	tr = tr.push(atomic.AddUint64(&bd.goalIDSeq, 1))
	bd.Env.Log.Info("check `goal`", `goal`, g.String(), `trace`, tr)
	if len(g.ResultOf()) == 0 {
		return nil
	}
	for _, act := range g.ResultOf() {
		for _, pre := range act.Premises() {
			if err := bd.buildGoal(ctx, pre, tr); err != nil {
				return err
			}
		}
	}

	log := bd.Env.Log
	bd.Env.Log = log.With("trace", tr.String())
	defer func() { bd.Env.Log = log }()
	_, err := bd.updateGoal(ctx, g)
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
	logCfg := qblog.DefaultConfig.Clone()
	logCfg.SetWriter(bd.Env.Err)
	bd.Env.Log = qblog.New(logCfg).Logger
	return restore, nil
}

func (bd *Builder) Do(ctx context.Context, a *Action, env *Env) error {
	prjs, err := Goals(a.Results(), true, Tangible, AType[*Project])
	if err != nil {
		return fmt.Errorf("project build result: %w", err)
	}
	for _, prj := range prjs {
		if err := bd.ProjectContext(ctx, prj.Artefact.(*Project)); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) WriteHash(h hash.Hash, a *Action, env *Env) (bool, error) {
	return false, nil // TODO good idea to hash build project operation
}
