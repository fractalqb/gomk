package gomk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"unsafe"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type updater struct {
	Env       *Env
	LogDir    string
	MkDirMode fs.FileMode

	bid       gomkore.BuildID // => updater must not be used concurrently
	goalIDSeq uint64
}

func (up *updater) updateGoal(ctx context.Context, g *Goal) (bool, error) {
	gid := uintptr(unsafe.Pointer(g))
	g.LockPreActions(gid)
	defer g.UnlockPreActions()

	chgs := g.CheckPreTimes()
	if len(chgs) == 0 {
		up.Env.Log.Info("`goal` already up-to-date", `goal`, g.String())
		return false, nil
	}
	up.Env.Log.Info("`goal` `requires` actions",
		`goal`, g.String(),
		`requires`, len(chgs),
	)

	var err error
	switch g.UpdateMode.Actions() {
	case UpdAllActions:
		err = up.updateAll(ctx, g, chgs)
	case UpdSomeActions:
		err = up.updateSome(ctx, g, chgs)
	case UpdAnyAction:
		err = up.updateAny(ctx, g, chgs)
	case UpdOneAction:
		if l := len(chgs); l > 1 {
			err = fmt.Errorf("%d change actions for update mode One in goal %s",
				l,
				g.String(),
			)
		} else {
			err = up.updateOne(ctx, g, chgs[0])
		}
	default:
		err = fmt.Errorf("illegal update mode actions: %d", g.UpdateMode.Actions())
	}
	return true, err
}

func (up *updater) updateAll(ctx context.Context, g *Goal, _ []int) error {
	if g.UpdateMode.Ordered() {
		for _, act := range g.ResultOf() {
			if preBID, err := act.RunContext(ctx, up.bid, up.Env); err != nil {
				return err
			} else if preBID == up.bid {
				return fmt.Errorf("action %s potentially ran out of order", act)
			} else if preBID > up.bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	} else {
		for _, act := range g.ResultOf() {
			if preBID, err := act.RunContext(ctx, up.bid, up.Env); err != nil {
				return err
			} else if preBID > up.bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	}
	return nil
}

func (up *updater) updateSome(ctx context.Context, g *Goal, chgs []int) error {
	if len(chgs) > 1 && g.UpdateMode.Ordered() {
		for _, idx := range chgs {
			act := g.PreAction(idx)
			if preBID, err := act.RunContext(ctx, up.bid, up.Env); err != nil {
				return err
			} else if preBID == up.bid {
				return fmt.Errorf("action %s potentially ran out of order", act)
			} else if preBID > up.bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	} else {
		for _, idx := range chgs {
			act := g.PreAction(idx)
			if preBID, err := act.RunContext(ctx, up.bid, up.Env); err != nil {
				return err
			} else if preBID > up.bid {
				return fmt.Errorf("action %s already run by younger build %d",
					act,
					preBID,
				)
			}
		}
	}
	return nil

}

func (up *updater) updateAny(ctx context.Context, g *Goal, chgs []int) error {
	done := -1
	for i, act := range g.ResultOf() {
		preBID := act.LastBuild()
		switch {
		case preBID > up.bid:
			return fmt.Errorf("action %s already run by younger build %d",
				act.String(),
				preBID,
			)
		case preBID == up.bid:
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
	_, err := g.PreAction(chgs[0]).RunContext(ctx, up.bid, up.Env)
	return err
}

func (up *updater) updateOne(ctx context.Context, g *Goal, chg int) error {
	for i, act := range g.ResultOf() {
		preBID := act.LastBuild()
		switch {
		case preBID > up.bid:
			return fmt.Errorf("action %s already run by younger build %d",
				act.String(),
				preBID,
			)
		case preBID == up.bid:
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
	_, err := g.PreAction(chg).RunContext(ctx, up.bid, up.Env)
	return err
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

func (bd *Builder) Describe(*Action, *Env) string {
	return "Build project" // TODO better description
}
