package gomkore

import (
	"context"
	"hash"
	"sync/atomic"
)

// An Action is something you can do in your [Project] to achieve at least one
// [Goal]. The actual implementation of the action is an [Operation]. An action
// without an operation is an "implicit" action, i.e. if all its premises are
// true, all results of the action are implicitly given.
type Action struct {
	Op          Operation
	IgnoreError bool
	premises    []*Goal
	results     []*Goal

	prj     *Project
	lockGID BuildID
	lastBID BuildID
}

func (a *Action) Project() *Project   { return a.prj }
func (a *Action) Premises() []*Goal   { return a.premises }
func (a *Action) Premise(i int) *Goal { return a.premises[i] }
func (a *Action) Results() []*Goal    { return a.results }
func (a *Action) Result(i int) *Goal  { return a.results[i] }

func (a *Action) LastBuild() BuildID { return a.lastBID }

// Must not run concurrently, see [Goal.LockPreActions] and [tryLock]
func (a *Action) Run(bid BuildID, env *Env) (BuildID, error) {
	return a.RunContext(context.Background(), bid, env)
}

// Must not run concurrently, see [Goal.LockPreActions] and [tryLock]
func (a *Action) RunContext(ctx context.Context, bid BuildID, env *Env) (BuildID, error) {
	if bid <= a.lastBID {
		return a.lastBID, nil
	}
	env.Log.Debug("run `action`", `action`, a.String())
	a.lastBID = bid
	if a.Op == nil {
		return 0, nil
	}
	if env == nil {
		env = DefaultEnv()
	}
	err := a.Op.Do(ctx, a, env)
	switch {
	case err == nil:
		return 0, nil
	case a.IgnoreError:
		env.Log.Warn("ignoring `action` `error`",
			`action`, a.String(),
			`error`, err,
		)
		return 0, nil
	}
	return 0, err
}

func (a *Action) String() string {
	switch {
	case a == nil:
		return "<nil:Action>/" + a.Project().Name(nil)
	case a.Op == nil:
		return "implicit:" + a.Project().Name(nil)
	}
	return a.Op.Describe(a, nil)
}

func (a *Action) WriteHash(h hash.Hash, env *Env) (bool, error) {
	if a.Op == nil {
		return false, nil
	}
	return a.Op.WriteHash(h, a, env)
}

func (a *Action) tryLock(byGID BuildID) (blockingGID BuildID) {
	if atomic.CompareAndSwapUint64(&a.lockGID, 0, byGID) {
		return 0
	}
	return atomic.LoadUint64(&a.lockGID)
}

func (a *Action) unlock() { atomic.StoreUint64(&a.lockGID, 0) }

type Operation interface {
	// The hints are optional
	Describe(actionHint *Action, envHint *Env) string
	Do(ctx context.Context, a *Action, env *Env) error
	WriteHash(h hash.Hash, a *Action, env *Env) (bool, error)
}

func updateConsistency(results []*Goal) error {
	for _, g := range results {
		for _, i := range results {
			if err := g.UpdateConsistency(i); err != nil {
				return err
			}
		}
	}
	return nil
}
