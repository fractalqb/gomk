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
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/qblog"
)

// TODO Add a "dry-run" option
type Builder struct {
	Env       *Env
	LogDir    string
	MkDirMode fs.FileMode

	gidSeq gomkore.BuildID // generate goal ID for a build
}

// Project builds all leafs in prj.
func (bd *Builder) Project(prj *Project) error {
	return bd.ProjectContext(context.Background(), prj)
}

// ProjectContext builds all leafs in prj.
func (bd *Builder) ProjectContext(ctx context.Context, prj *Project) error {
	start := time.Now()
	bid := prj.LockBuild()
	defer prj.Unlock()
	if bd.Env == nil {
		bd.Env = DefaultEnv()
	}
	oldEnv, err := bd.prepLogDir(bid)
	if err != nil {
		return err
	}
	defer func() { bd.restoreEnv(oldEnv) }()
	tr := startTrace(bid)
	bd.Env.Log.Info("`build` `project` in `dir`",
		`build`, fmt.Sprintf("@%d", bid),
		`project`, prj.String(),
		`dir`, prj.Dir,
	)
	leafs := prj.Leafs()
	bd.gidSeq = 0
	for _, leaf := range leafs {
		if err := bd.buildGoal(ctx, leaf, tr); err != nil {
			return err
		}
	}
	bd.Env.Log.Info("`build` of `project` `took`",
		`build`, fmt.Sprintf("@%d", bid),
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
	bd.gidSeq = 0
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

func (bd *Builder) nextGoalID() any { return atomic.AddUint64(&bd.gidSeq, 1) }

func (bd *Builder) buildGoal(ctx context.Context, g *Goal, tr *trace) error {
	bid := g.LockBuild(bd.nextGoalID)
	if bid == 0 {
		return nil
	}
	gid := g.BuildInfo().(gomkore.BuildID) // my goal ID
	tr = tr.push(gid)
	defer g.Unlock()
	bd.Env.Log.Info("check `goal`", `goal`, g.String(), `trace`, tr)
	if len(g.ResultOf) == 0 {
		return nil
	}
	for _, act := range g.ResultOf {
		for _, pre := range act.Premises() {
			if err := bd.buildGoal(ctx, pre, tr); err != nil {
				return err
			}
		}
	}

	update, chgs := bd.checkPreTimes(g)
	if !update {
		bd.Env.Log.Info("`goal` already up-to-date",
			`goal`, g.String(),
			`trace`, tr,
		)
		return nil
	}
	bd.Env.Log.Info("`goal` `requres` actions",
		`goal`, g.String(),
		`requres`, len(chgs),
		`trace`, tr,
	)

	g.LockPreActions(gid)
	defer g.UnlockPreActions()

	log := bd.Env.Log
	bd.Env.Log = log.With("trace", tr.String())
	defer func() { bd.Env.Log = log }()

	var err error
	switch g.UpdateMode.Actions() {
	case UpdAllActions:
		err = bd.updateAll(ctx, bid, g, chgs)
	case UpdSomeActions:
		err = bd.updateSome(ctx, bid, g, chgs)
	case UpdAnyAction:
		err = bd.updateAny(ctx, bid, g, chgs)
	case UpdOneAction:
		if l := len(chgs); l > 1 {
			err = fmt.Errorf("%d change actions for update mode One in goal %s",
				l,
				g.String(),
			)
		} else {
			err = bd.updateOne(ctx, bid, g, chgs[0])
		}
	default:
		err = fmt.Errorf("illegal update mode actions: %d", g.UpdateMode.Actions())
	}
	return err
}

func (bd *Builder) updateAll(ctx context.Context, bid gomkore.BuildID, g *Goal, _ []int) error {
	if g.UpdateMode.Ordered() {
		for _, act := range g.ResultOf {
			if preBID, err := act.RunContext(ctx, bid, bd.Env); err != nil {
				return err
			} else if preBID == bid {
				return fmt.Errorf("action %s potentially ran out of order", act)
			} else if preBID > bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	} else {
		for _, act := range g.ResultOf {
			if preBID, err := act.RunContext(ctx, bid, bd.Env); err != nil {
				return err
			} else if preBID > bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	}
	return nil
}

func (bd *Builder) updateSome(ctx context.Context, bid gomkore.BuildID, g *Goal, chgs []int) error {
	if len(chgs) > 1 && g.UpdateMode.Ordered() {
		for _, idx := range chgs {
			act := g.ResultOf[idx]
			if preBID, err := act.RunContext(ctx, bid, bd.Env); err != nil {
				return err
			} else if preBID == bid {
				return fmt.Errorf("action %s potentially ran out of order", act)
			} else if preBID > bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	} else {
		for _, idx := range chgs {
			act := g.ResultOf[idx]
			if preBID, err := act.RunContext(ctx, bid, bd.Env); err != nil {
				return err
			} else if preBID > bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	}
	return nil

}

func (bd *Builder) updateAny(ctx context.Context, bid gomkore.BuildID, g *Goal, chgs []int) error {
	done := -1
	for i, act := range g.ResultOf {
		preBID := act.LastBuild()
		switch {
		case preBID > bid:
			return fmt.Errorf("action %s already run by younger build %d",
				act.String(),
				preBID,
			)
		case preBID == bid:
			if slices.Index(chgs, i) < 0 {
				return fmt.Errorf(
					"goal %s with update mode Any involved by inconsistent action",
					g.String(),
				)
			} else if done < 0 {
				done = i
			} else {
				return fmt.Errorf(
					"goal %s with update mode Any already ran more than one action",
					g.String(),
				)
			}
		}
	}
	if done >= 0 {
		return nil
	}
	_, err := g.ResultOf[chgs[0]].RunContext(ctx, bid, bd.Env)
	return err
}

func (bd *Builder) updateOne(ctx context.Context, bid gomkore.BuildID, g *Goal, chg int) error {
	for i, act := range g.ResultOf {
		preBID := act.LastBuild()
		switch {
		case preBID > bid:
			return fmt.Errorf("action %s already run by younger build %d",
				act.String(),
				preBID,
			)
		case preBID == bid:
			if i == chg {
				return nil
			} else {
				return fmt.Errorf(
					"goal %s with update mode Any involved by inconsistent action",
					g.String(),
				)
			}
		}
	}
	_, err := g.ResultOf[chg].RunContext(ctx, bid, bd.Env)
	return err
}

// need update if there are no pre goals or if timestamps indicate an update
func (bd *Builder) checkPreTimes(g *Goal) (update bool, chgs []int) {
	// TODO Consistency for concurrent builds
	update = true
	gaTS := g.Artefact.StateAt(g.Project())
	for actIdx, act := range g.ResultOf {
		if gaTS.IsZero() {
			chgs = append(chgs, actIdx)
			continue
		}
	PREMISE_LOOP:
		for _, pre := range act.Premises() {
			update = false
			preTS := pre.Artefact.StateAt(g.Project())
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
	return update || len(chgs) > 0, chgs
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

func (bd *Builder) prepLogDir(bid gomkore.BuildID) (restore *Env, err error) {
	if bd.LogDir == "" {
		return bd.Env, nil
	}
	bdir := fmt.Sprintf("%s.%d", time.Now().Format("060102-150405"), bid)
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

type trace struct {
	p  *trace
	id gomkore.BuildID
}

func startTrace(id gomkore.BuildID) *trace { return &trace{id: id} }

func (t *trace) push(id gomkore.BuildID) *trace { return &trace{p: t, id: id} }

// func (t *trace) pop() *trace { return t.p }

// must t != nil
func (t *trace) String() string {
	var sb strings.Builder
	if t.p != nil {
		fmt.Fprint(&sb, t.id)
		for t = t.p; t.p != nil; t = t.p {
			fmt.Fprintf(&sb, ">%d", t.id)
		}
	}
	fmt.Fprintf(&sb, "@%d", t.id)
	return sb.String()
}
