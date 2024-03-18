package gomkore

import (
	"fmt"
	"slices"
	"unsafe"
)

type updater struct {
	trace *Trace
	env   *Env
	bid   BuildID // => updater must not be used concurrently
}

func (up *updater) Trace() *Trace { return up.trace }

func (up *updater) updateGoal(tr *Trace, g *Goal) (bool, error) {
	gid := uintptr(unsafe.Pointer(g))
	g.LockPreActions(gid)
	defer g.UnlockPreActions()

	chgs, err := g.CheckPreTimes(tr)
	if err != nil {
		return false, err
	}
	if len(chgs) == 0 {
		tr.goalUpToDate(g)
		return false, nil
	}
	tr.goalNeedsActions(g, len(chgs))

	switch g.UpdateMode.Actions() {
	case UpdAllActions:
		err = up.updateAll(tr, g, chgs)
	case UpdSomeActions:
		err = up.updateSome(tr, g, chgs)
	case UpdAnyAction:
		err = up.updateAny(tr, g, chgs)
	case UpdOneAction:
		if l := len(chgs); l > 1 {
			err = fmt.Errorf("%d change actions for update mode One in goal %s",
				l,
				g.String(),
			)
		} else {
			err = up.updateOne(tr, g, chgs[0])
		}
	default:
		err = fmt.Errorf("illegal update mode actions: %d", g.UpdateMode.Actions())
	}
	return true, err
}

func (up *updater) updateAll(tr *Trace, g *Goal, _ []int) error {
	switch len(g.ResultOf()) {
	case 0:
		return nil
	case 1:
		act := g.PreAction(0)
		preBID, err := act.Run(tr, up.env)
		if err != nil {
			return err
		} else if preBID > up.bid {
			return fmt.Errorf("action %s already run by younger build %d",
				act,
				preBID,
			)
		}
		return nil
	}
	if g.UpdateMode.Ordered() {
		for _, act := range g.ResultOf() {
			if preBID, err := act.Run(tr, up.env); err != nil {
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
			if preBID, err := act.Run(tr, up.env); err != nil {
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

func (up *updater) updateSome(tr *Trace, g *Goal, chgs []int) error {
	if len(chgs) > 1 && g.UpdateMode.Ordered() {
		for _, idx := range chgs {
			act := g.PreAction(idx)
			if preBID, err := act.Run(tr, up.env); err != nil {
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
			if preBID, err := act.Run(tr, up.env); err != nil {
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

func (up *updater) updateAny(tr *Trace, g *Goal, chgs []int) error {
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
	_, err := g.PreAction(chgs[0]).Run(tr, up.env)
	return err
}

func (up *updater) updateOne(tr *Trace, g *Goal, chg int) error {
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
	_, err := g.PreAction(chg).Run(tr, up.env)
	return err
}
