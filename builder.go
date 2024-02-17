package gomk

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"git.fractalqb.de/fractalqb/qblog"
)

// TODO Add a "dry-run" option
type Builder struct {
	Env       *Env
	LogDir    string
	MkDirMode fs.FileMode

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
	oldEnv, err := bd.prepLogDir()
	if err != nil {
		return err
	}
	defer func() { bd.restoreEnv(oldEnv) }()
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

// TODO LogDir
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
	chgs := bd.checkPreTimes(g)
	if len(chgs) == 0 {
		bd.Env.Log.Debug("`goal` already up-to-date", `goal`, g.String())
		return nil
	}
	var err error
	switch g.UpdateMode.Actions() {
	case UpdAllActions:
		err = bd.updateAll(ctx, g, chgs)
	case UpdSomeActions:
		err = bd.updateSome(ctx, g, chgs)
	case UpdAnyAction:
		err = bd.updateAny(ctx, g, chgs)
	case UpdOneAction:
		err = bd.updateOne(ctx, g, chgs)
	default:
		err = fmt.Errorf("illegal update mode actions: %d", g.UpdateMode.Actions())
	}
	if err != nil {
		return err
	}
	return nil
}

func (bd *Builder) updateAll(ctx context.Context, g *Goal, chgs []int) error {
	for _, a := range g.ResultOf {
		if err := a.RunContext(ctx, bd.Env); err != nil {
			return err
		}
	}
	return nil
}

func (bd *Builder) updateSome(ctx context.Context, g *Goal, chgs []int) error {
	for _, idx := range chgs {
		if err := g.ResultOf[idx].RunContext(ctx, bd.Env); err != nil {
			return err
		}
	}
	return nil

}

func (bd *Builder) updateAny(ctx context.Context, g *Goal, chgs []int) error {
	if len(chgs) > 0 {
		err := g.ResultOf[chgs[0]].RunContext(ctx, bd.Env)
		return err
	}
	return nil
}

func (bd *Builder) updateOne(ctx context.Context, g *Goal, chgs []int) error {
	switch len(chgs) {
	case 0:
		return nil
	case 1:
		err := g.ResultOf[chgs[0]].RunContext(ctx, bd.Env)
		return err
	}
	return fmt.Errorf("%d update actions for goal %s", len(chgs), g.Name())
}

func (bd *Builder) checkPreTimes(g *Goal) (chgs []int) {
	gaTS := g.Artefact.StateAt()
	for actIdx, act := range g.ResultOf {
		if gaTS.IsZero() {
			chgs = append(chgs, actIdx)
			continue
		}
	PREMISE_LOOP:
		for _, pre := range act.Premises {
			preTS := pre.Artefact.StateAt()
			switch {
			case preTS.IsZero():
				chgs = append(chgs, actIdx)
				break PREMISE_LOOP
			case gaTS.Before(preTS):
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

type stackedWriter struct {
	top, tail io.Writer
}

func (w stackedWriter) Write(p []byte) (n int, err error) {
	n1, err := w.top.Write(p)
	n2, err2 := w.tail.Write(p)
	if err2 != nil {
		if err == nil {
			err = err2
		} else {
			err = errors.Join(err, err2)
		}
	}
	return n1 + n2, err
}

func (w stackedWriter) Close() error {
	if c, ok := w.top.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
